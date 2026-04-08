# Quotes Reader / Collector (Go)

Реализована только Go-часть из ТЗ:
- `Quotes Reader / Collector`
- без `MOEX HTTP Fetcher`

Сервис:
- читает бинарный поток TickFrame v1 из устройства драйвера;
- публикует котировки в Valkey Pub/Sub;
- пишет raw ticks в ClickHouse.

Поддерживаются два режима входного потока:
- `COLLECTOR_INPUT_MODE=vtp` — ожидает реальные `TickFrame v1`.
- `COLLECTOR_INPUT_MODE=random` — преобразует произвольные байты устройства в синтетические тики.

Для учебного проекта по умолчанию используется `random`.
Дефолтный `COLLECTOR_DEVICE_PATH` — `/dev/urandom`.
Чтобы замедлить поток, используйте `COLLECTOR_PUBLISH_DELAY` (например, `10ms`).

Если устройство/pipe ещё не создано, collector будет ждать и переоткрывать путь
(`COLLECTOR_DEVICE_OPEN_RETRY`, по умолчанию `1s`).

## Структура

- `cmd/quotes-collector` — точка входа.
- `internal/vtp` — формат TickFrame v1, CRC32, stream parser.
- `internal/randomticks` — преобразование случайного потока в тики.
- `internal/collector` — цикл чтения устройства и dispatch в sink-и.
- `internal/sink` — Valkey Pub/Sub, ClickHouse HTTP, stdout.
- `.env.example` — параметры запуска.

## Формат TickFrame v1

- Magic: `0x4D 0x58` (`MX`)
- Version: `0x01`
- MsgType: `0x01` (tick)
- SeqNo: `uint64`
- TsUnixMs: `uint64`
- InstrumentId: `uint64`
- Price: `int64` (в копейках)
- Qty: `uint64` (лоты)
- Side: `uint8` (`0/1/2`)
- CRC32: `uint32` (IEEE, по первым 45 байтам)

Кодирование в текущей реализации: **big-endian**.

## Запуск

```bash
cp .env.example .env
set -a && source .env && set +a
go run ./cmd/quotes-collector
```

## Docker Compose

Поднимает полный локальный стенд:
- `valkey`
- `clickhouse`
- `quotes-collector`

```bash
docker compose up --build -d
docker compose logs -f quotes-collector
```

Остановить:

```bash
docker compose down
```

Примечание:
- в compose по умолчанию используется `COLLECTOR_INPUT_MODE=random` и `COLLECTOR_DEVICE_PATH=/dev/urandom`;
- можно переключить на `vtp` и ваш device, если внешний источник пишет валидные `TickFrame`;
- `MOEX HTTP Fetcher` не добавлялся (как и требовалось).

## Тесты

```bash
go test ./...
```
