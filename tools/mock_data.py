from dataclasses import dataclass
import random
from datetime import datetime, timedelta
import asyncio
import httpx


@dataclass
class TrialData:
    trial_id: int
    trial_start_time: str
    trial_end_time: str
    result: bool
    correct_rate: float  # 添加正确率字段


class TrialStats:
    def __init__(self):
        self.total_trials = 0
        self.successful_trials = 0

    def update(self, result: bool):
        self.total_trials += 1
        if result:
            self.successful_trials += 1

    @property
    def accuracy(self) -> float:
        if self.total_trials == 0:
            return 0.0
        return (
            self.successful_trials / self.total_trials
        )  # 返回原始 float 值，不是百分比

    def __str__(self) -> str:
        return f"Accuracy: {self.accuracy:.4f} ({self.successful_trials}/{self.total_trials})"


async def send_data(client: httpx.AsyncClient, data: TrialData) -> bool:
    try:
        response = await client.post(
            "http://localhost:9012/api/broadcast/data/default",
            json={
                "trial_id": data.trial_id,
                "start_time": data.trial_start_time,
                "end_time": data.trial_end_time,
                "result": data.result,
                "correct_rate": data.correct_rate,  # 发送正确率数据
            },
            timeout=10.0,
        )
        response.raise_for_status()
        print(f"Data #{data.trial_id} sent successfully!")
        return True
    except httpx.HTTPError as e:
        print(f"Failed to send data #{data.trial_id}: {str(e)}")
        return False


async def gen_data(
    count: int, last_end_time: datetime | None, current_accuracy: float
) -> tuple[TrialData, datetime, int]:
    start_time = (
        last_end_time + timedelta(seconds=10) if last_end_time else datetime.now()
    )

    duration = random.randint(1, 10)
    duration_time = timedelta(seconds=duration)
    end_time = start_time + duration_time

    result = random.choice([True, False])

    return (
        TrialData(
            trial_id=count,
            trial_start_time=start_time.isoformat(),
            trial_end_time=end_time.isoformat(),
            result=result,
            correct_rate=current_accuracy,  # 包含当前正确率
        ),
        end_time,
        duration,
    )


async def main():
    count = 1
    last_end_time: datetime | None = None
    stats = TrialStats()

    async with httpx.AsyncClient(
        http2=True,
        timeout=httpx.Timeout(10.0),
        limits=httpx.Limits(max_connections=100),
    ) as client:
        while True:
            # 先获取当前正确率
            current_accuracy = stats.accuracy

            # 生成试验数据，传入当前正确率
            trial_data, last_end_time, duration = await gen_data(
                count, last_end_time, current_accuracy
            )

            # 更新统计数据（基于本次试验的结果）
            stats.update(trial_data.result)

            print(f"Trial #{trial_data.trial_id}:")
            print(f"  Start: {trial_data.trial_start_time}")
            print(f"  End:   {trial_data.trial_end_time}")
            print(f"  Duration: {duration} s")
            print(f"  Result: {'Success' if trial_data.result else 'Failure'}")
            print(f"  {stats}")
            print("-" * 50)

            await send_data(client, trial_data)

            count += 1
            await asyncio.sleep(duration)


if __name__ == "__main__":
    asyncio.run(main())
