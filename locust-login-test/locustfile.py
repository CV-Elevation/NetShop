"""
登录稳定性压测 - Locust 脚本
测试范围：
  - Task 1: token 有效 → 拦截器校验 → 放行（模拟已登录用户）
  - Task 2: token 缺失 → 拦截器拦截 → 返回 401/302（验证拦截行为稳定）

运行方式：
  locust -f locustfile.py --host http://localhost:8080
  然后打开 http://localhost:8089 配置并发数
"""

from locust import HttpUser, task, between, events
from locust.exception import StopUser
import random
import logging

logger = logging.getLogger(__name__)

# ────────────────────────────────────────────
# 配置区：填入你的有效 token 列表
# 可以手动登录后从浏览器 Cookie 里复制
# ────────────────────────────────────────────
VALID_TOKENS = [
    # "your_token_1_here",
    # "your_token_2_here",
    # "your_token_3_here",
    "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0b2tlbl90eXBlIjoiYWNjZXNzIiwidWlkIjoiOGFlYjhiY2UtMDE4MC00OTFhLTlhMDItMzk4ZGY4ZWI4ZTgxIiwiZW1haWwiOiIyMjQxMDQ1NzYyQHFxLmNvbSIsIm5pY2tuYW1lIjoiS3VvWiIsInN1YiI6IjhhZWI4YmNlLTAxODAtNDkxYS05YTAyLTM5OGRmOGViOGU4MSIsImV4cCI6MTc3NDY4OTUyNSwibmJmIjoxNzc0NjgyNjI1LCJpYXQiOjE3NzQ2ODI2MjV9.3rJuCChYzVZIi-jVexq78tO7Mbr6vOezklaE5W4fvE0",
]

COOKIE_NAME   = "access_token"
PROTECTED_URL = "/"


# ────────────────────────────────────────────
# 统计收集：自定义指标
# ────────────────────────────────────────────
class LoginStats:
    def __init__(self):
        self.valid_success   = 0   # token 有效且被正确放行
        self.valid_rejected  = 0   # token 有效但被错误拒绝（bug）
        self.notoken_blocked = 0   # 无 token 被正确拦截
        self.notoken_pass    = 0   # 无 token 被错误放行（bug）

stats = LoginStats()


@events.quitting.add_listener
def on_quit(environment, **kwargs):
    """压测结束时打印自定义统计"""
    print("\n========== 登录稳定性测试报告 ==========")
    print(f"  ✅ 有效token正确放行:   {stats.valid_success}")
    print(f"  ❌ 有效token错误拒绝:   {stats.valid_rejected}  ← 应为 0")
    print(f"  ✅ 无token正确拦截:     {stats.notoken_blocked}")
    print(f"  ❌ 无token错误放行:     {stats.notoken_pass}    ← 应为 0")

    total_valid = stats.valid_success + stats.valid_rejected
    if total_valid > 0:
        success_rate = stats.valid_success / total_valid * 100
        print(f"\n  登录放行成功率: {success_rate:.2f}%  (目标 > 99.9%)")
    print("==========================================\n")


# ────────────────────────────────────────────
# 场景一：持有有效 token 的已登录用户
# ────────────────────────────────────────────
class LoggedInUser(HttpUser):
    """
    模拟已登录用户持续访问受保护接口
    重点验证：token 校验链路在高并发下的稳定性
    """
    wait_time = between(0.5, 2)   # 每次请求间隔 0.5~2s，模拟真实节奏
    weight    = 7                  # 占 70% 并发（主要场景）

    def on_start(self):
        if not VALID_TOKENS:
            logger.warning(
                "VALID_TOKENS 为空！请在脚本顶部填入有效 token。"
                "当前将跳过 LoggedInUser 场景。"
            )
            raise StopUser()
        # 每个虚拟用户随机领取一个 token，模拟不同 session
        self.token = random.choice(VALID_TOKENS)
        self.client.cookies.set(COOKIE_NAME, self.token)

    @task
    def access_protected_route(self):
        with self.client.post(
            PROTECTED_URL,
            allow_redirects=False,   # 不跟随跳转，直接看原始状态码
            catch_response=True,
            name="[有效token] 访问受保护接口",
        ) as resp:
            # 放行：200 或业务正常响应
            if resp.status_code in (200, 204):
                stats.valid_success += 1
                resp.success()

            # 被重定向到 GitHub 登录：token 有效却被拦截，属于 bug
            elif resp.status_code in (301, 302, 307, 308):
                location = resp.headers.get("Location", "")
                stats.valid_rejected += 1
                resp.failure(f"有效token被拦截，重定向至: {location}")

            # 401/403：token 校验失败
            elif resp.status_code in (401, 403):
                stats.valid_rejected += 1
                resp.failure(f"有效token被拒绝，状态码: {resp.status_code}")

            else:
                resp.failure(f"未预期状态码: {resp.status_code}")


# ────────────────────────────────────────────
# 场景二：没有 token 的未登录用户
# ────────────────────────────────────────────
class AnonymousUser(HttpUser):
    """
    模拟未登录用户访问受保护接口
    重点验证：拦截器在高并发下是否稳定拦截，不会漏放
    """
    wait_time = between(1, 3)
    weight    = 3                  # 占 30% 并发

    @task
    def access_without_token(self):
        with self.client.get(
            PROTECTED_URL,
            allow_redirects=False,
            catch_response=True,
            name="[无token] 访问受保护接口",
        ) as resp:
            # 正确行为：拦截并跳转 GitHub 登录，或返回 401
            if resp.status_code in (301, 302, 307, 308, 401, 403):
                stats.notoken_blocked += 1
                resp.success()   # 这是预期行为，标记为成功

            # 被错误放行：严重 bug
            elif resp.status_code in (200, 204):
                stats.notoken_pass += 1
                resp.failure("无token请求被错误放行！")

            else:
                resp.failure(f"未预期状态码: {resp.status_code}")
