# Solar

[English](README.md) | **Русский**

Solar — сервер Minecraft Classic и ClassiCube с открытым исходным кодом,
написанный на Go. Проект предоставляет компактное и тестируемое серверное ядро
с поддержкой Classic Protocol Extension (CPE), несколькими мирами, физикой
блоков, модерацией и двумя вариантами плагинов.

> [!WARNING]
> Solar находится на стадии pre-alpha. Конфигурация, форматы хранения, команды
> и API плагинов могут меняться без обратной совместимости. Делайте резервные
> копии данных перед обновлением и проверяйте изменения до запуска на публичном
> сервере.

## Почему Solar?

- **Ориентация на Classic и ClassiCube.** Solar реализует Classic-протокол и
  большой набор расширений CPE, а не современные протоколы Java или Bedrock.
- **Расширяемость.** Доступны API плагинов на Go и опциональная поддержка Lua.
- **Игровая логика в духе MCGalaxy.** Реализованы знакомые команды, ранги,
  информационные блоки, порталы, двери и физика блоков.
- **Защита эксплуатации.** Есть атомарные сохранения миров, TCP-таймауты,
  backpressure, аутентификация, антиспам, структурированные логи, корректное
  завершение работы и тестирование с Go race detector.
- **Инструменты разработки.** В репозитории есть синтетические Classic-клиенты
  и воспроизводимые сценарии нагрузочного тестирования.

## Чем Solar не является

Solar не является сервером ванильного выживания и не реализует современные
протоколы Minecraft Java Edition или Bedrock Edition. В нём нет ванильных
мобов, инвентаря, крафта, редстоуна и прогрессии выживания. Solar подходит для
строительного сервера Minecraft Classic или ClassiCube с собственной игровой
логикой.

Solar — независимый проект. Он не связан с Mojang Studios, Microsoft,
ClassiCube или MCGalaxy и не одобрен ими.

## Возможности

- Вход Classic-клиентов, загрузка мира, движение, чат и обновление сущностей
- Согласование CPE, включая FastMap, расширенный список игроков, пользовательские
  блоки, окружение, выделения, plugin messages и расширенную телепортацию
- Несколько загружаемых миров, бинарный формат `.swld` и автосохранение
- Семейства генераторов Simple, Classic, fCraft и heightmap
- Отдельная для каждого мира физика воды, лавы, песка, гравия, травы, листвы,
  огня, TNT и дверей
- Ранги, права, whitelist, баны, mute, freeze, hide, AFK и антиспам
- Команды рисования, буфер обмена, undo/redo, история BlockDB, message blocks и
  порталы
- Английские и русские сообщения с выбором языка для каждого игрока
- Плагины на Go на поддерживаемых платформах и опциональные Lua-плагины
- Встроенные сценарии нагрузки `idle`, `chat`, `move`, `blocks` и `mixed`

Список основных команд находится в
[`docs/COMMAND_MATRIX.md`](docs/COMMAND_MATRIX.md). Проверки готовности к
релизу перечислены в [`docs/GAMEPLAY_MATRIX.md`](docs/GAMEPLAY_MATRIX.md).

## Требования

- Go 1.24 или новее
- Git и поддерживаемая операционная система для сборки из исходников
- Клиент, совместимый с Minecraft Classic или ClassiCube

Go-плагины `.so` загружаются только на Linux и macOS. Плагин должен быть собран
той же версией Go и с совместимыми версиями зависимостей, что и сервер.

## Быстрый запуск

```sh
git clone https://github.com/solar-mc/solar.git
cd solar
make build
./bin/solar start --config configs/server.toml
```

Сервер можно запустить напрямую из исходников:

```sh
go run ./cmd/solar start --config configs/server.toml
```

Подключитесь клиентом к `127.0.0.1:25565`. При первом запуске Solar создаст
каталоги данных и сгенерирует основной мир, если сохранённого мира ещё нет.

## Конфигурация

Пример конфигурации находится в
[`configs/server.toml`](configs/server.toml). В нём описаны все поддерживаемые
параметры; используйте этот файл как отправную точку.

```toml
listen = ":25565"
data_dir = "data"
max_players = 128
autosave_interval = "60s"
default_generator = "Classic"
server_name = "Solar"

[storage]
main_world_name = "main"

[language]
default = "ru"
file = "configs/language.toml"

[auth]
enabled = false
salt = ""

[heartbeat]
enabled = false
public = false
```

Важные группы параметров:

| Область | Параметры |
| --- | --- |
| Сеть | `listen`, `max_players`, таймауты и очереди в `[network]` |
| Миры | `default_generator`, `[world]`, `[storage]`, `autosave_interval` |
| Доступ | `operators`, `[auth]`, `[heartbeat]`, whitelist в `[player]` |
| Защита | `[antispam]` и `[afk]` |
| Расширения | `[cpe]`, `[plugins]` и `[lua]` |
| Эксплуатация | `[log]` и `[debug].pprof_address` |

Операторов также можно задать через `SOLAR_OPERATORS="alice,bob"`.

### Публичный сервер

Перед публикацией сервера:

1. Включите `[auth]` для проверки Classic-имён через `mppass`.
2. Включите и настройте `[heartbeat]`, если сервер должен появиться в списке
   серверов ClassiCube.
3. Сделайте настроенный TCP-порт доступным из интернета.
4. Разместите `data/` на постоянном хранилище и регулярно создавайте резервные
   копии.
5. Запустите production gate и проверьте сервер настоящим клиентом ClassiCube.

## Серверные команды

Введите `/help` в игре, чтобы увидеть команды, доступные вашему рангу. Основные
группы:

- Миры: `/newlvl`, `/load`, `/unload`, `/goto`, `/save`, `/map`
- Модерация: `/kick`, `/ban`, `/whitelist`, `/mute`, `/freeze`, `/hide`
- Телепортация: `/tp`, `/spawn`, `/back`, `/tpa`, `/tpaccept`, `/tpdeny`
- Строительство: `/cuboid`, `/line`, `/sphere`, `/fill`, `/copy`, `/paste`
- Интерактивные блоки: `/mb`, `/portal`, `/door`
- История и восстановление: `/blockdb`, `/blockundo`, `/undo`, `/redo`
- Ранги и информация: `/setrank`, `/rankinfo`, `/viewranks`, `/whois`

Основные команды и их права приведены в
[матрице команд](docs/COMMAND_MATRIX.md).

## Плагины

### Плагины на Go

Публичный API находится в [`plugin/`](plugin/). Минимальный пример динамически
загружаемого плагина расположен в
[`plugins/soplug`](plugins/soplug).

Команды плагинов могут объявлять алиасы, справку и минимальный ранг через
`plugin.CommandSpec`. API игрока предоставляет ранг, права строительства и
интерактивную разметку блоков через `SelectBlocks`; глобальные файлы плагина
следует хранить в каталоге, возвращаемом `Server.PluginDataDir`.

```sh
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/soplug.so ./plugins/soplug
```

После сборки включите `[plugins]` в `configs/server.toml`. Пересобирайте плагины
после изменения версии Go или зависимостей сервера.

Наборы [`plugins/mcgalaxy`](plugins/mcgalaxy) добавляют команды MCGalaxy, не
включая их в ядро сервера. Каждая категория команд является независимым
плагином; все три можно загрузить одновременно:

```sh
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-chat.so ./plugins/mcgalaxy/chat
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-cpe.so ./plugins/mcgalaxy/cpe
go build -tags=plugin -buildmode=plugin \
  -o data/plugins/mcgalaxy-economy.so ./plugins/mcgalaxy/economy
```

Плагин чата предоставляет `/roll`, `/8ball`, `/high5`, `/hug`, `/eat`, `/say`,
`/clear` и `/color`. CPE-плагин предоставляет `/reachdistance`. Плагин
экономики предоставляет `/balance`, `/pay`, `/give`, `/take` и `/economy`;
его состояние атомарно сохраняется в приватном каталоге.

### Плагины на Lua

Поддержка Lua не входит в стандартный бинарный файл. Соберите её явно:

```sh
go build -tags=lua -o bin/solar ./cmd/solar
mkdir -p data/lua
cp plugins/example.lua data/lua/example.lua
```

Перед запуском включите `[lua]` в конфигурации. Пример
[`plugins/example.lua`](plugins/example.lua) показывает события, команды,
операции с игроками, миры и запланированные задачи.

## Нагрузочное тестирование

Solar содержит встроенный генератор синтетических Classic-клиентов:

```sh
./bin/solar loadtest \
  --address 127.0.0.1:25565 \
  --clients 64 \
  --duration 30s \
  --scenario mixed \
  --cpe
```

При включённой аутентификации добавьте `--auth-salt <salt>`.

## Разработка

```sh
make test          # модульные и интеграционные тесты
make test-race     # тесты с Go race detector
make vet           # статические проверки
make lint          # golangci-lint
make build         # сборка bin/solar
make production-gate
```

Production gate запускает тесты, `go vet`, race detector, смешанную CPE-нагрузку,
health checks и, если не отключено, smoke-тест настоящим клиентом ClassiCube.

Структура репозитория:

| Путь | Назначение |
| --- | --- |
| `cmd/solar/` | Точка входа CLI |
| `internal/protocol/classic/` | Реализация Classic и CPE |
| `internal/server/` | Жизненный цикл, миры, физика и сохранение |
| `internal/command/` | Реестр и встроенные команды |
| `internal/blocks/` | Определения блоков, физика, история и рисование |
| `internal/generator/` | Встроенные генераторы миров |
| `plugin/` | Публичный Go API плагинов |
| `internal/loadtest/` | Синтетические Classic-клиенты |
| `third_party/` | Исходники и ресурсы upstream-проектов для сверки |

Мы принимаем изменения от сообщества. Прочитайте
[`CONTRIBUTING.md`](CONTRIBUTING.md), делайте небольшие целевые изменения и
добавляйте тесты для нового поведения.

## Лицензия

Solar распространяется по [лицензии MIT](LICENSE).

Исходники и ресурсы в `third_party/` сохраняют лицензии соответствующих
upstream-проектов.
