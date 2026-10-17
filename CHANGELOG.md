# Changelog

Формат основан на [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/).

## [Unreleased]

### Добавлено

- **Connect-шлюз** `grpcconnect` (`soma.WithGRPCConnect`, порт 1481): принимает Connect, gRPC-Web
  и gRPC (https://connectrpc.com) и проксирует запросы в gRPC-сервер. Работает рядом с grpc-gateway
  и не требует кодогенерации: методы берутся из дескрипторов зарегистрированных сервисов
  (`vanguard.ForwardService`), служебные `grpc.*` (health, reflection) наружу не отдаются.
  Запросы приходят в gRPC-сервер обычным gRPC, поэтому его интерцепторы выполняются как обычно;
  заголовки и metadata пробрасываются в обе стороны (включая `Set-Cookie`).
- `grpcconnect`: `WithPort`, `WithShutdownTimeout`, `WithHandlerOptions` (опции `connecthttp.Mount`),
  `WithTLSConfig` (TLS к gRPC-серверу), `WithLogger`.
- `grpcserver.Server.GetServiceInfo()`.
- `soma.ErrConnectRequiresGRPCServer`.

### Изменено

- Минимальная версия Go — 1.26 (требование `connectrpc.com/connect/v2` и `connectrpc.com/vanguard`).
- `google.golang.org/grpc` v1.81.1 → v1.83.2.

## [1.1.15]

Изменения относительно `v1.1.14`. Релиз содержит несовместимые изменения API — см. раздел
[«Миграция»](#миграция-с-v1114).

### Исправлено

#### Жизненный цикл

- Компоненты (планировщик, gRPC-сервер, gateway, профайлер) запускались всегда, независимо от опций
  `With*`: `Default()` заполнял их конфиги. Теперь каждый компонент включается только своей опцией.
- Ошибка `Prepare` не освобождала ресурсы: у уже подготовленных процессов не вызывался `Shutdown`,
  а ресурсы из `WithClosers` не закрывались. Теперь подготовленные процессы откатываются в обратном
  порядке, после чего закрываются ресурсы.
- Остановка могла зависнуть навсегда: ожидание завершения `Run` ничем не ограничивалось. Теперь оно
  ограничено `WithShutdownTimeout`.
- Ресурсы из `WithClosers` получали тот же контекст, что и процессы, и при его истечении пропускались
  целиком. Теперь у закрытия ресурсов отдельный бюджет `WithCloseTimeout`, а зависшая функция
  закрытия не блокирует остановку дольше этого бюджета.
- Если все процессы завершались сами, `Run` висел до сигнала. Теперь приложение переходит к остановке.
- Сигналы: подписка на SIGINT/SIGTERM появлялась только после `Prepare` и никогда не снималась,
  а повторный Ctrl+C во время зависшей остановки терялся. Теперь подписка оформляется до `Prepare`
  и снимается в начале остановки: повторный сигнал завершает процесс сразу.
- Ошибки `Shutdown` только логировались и не попадали в результат `Run`. Теперь `Run` возвращает
  объединение ошибок всех фаз, а в каждой ошибке указано имя процесса.
- При падении процесса в лог писалось вводящее в заблуждение «context canceled». Теперь причина
  остановки логируется как есть.
- `Validate` никогда не срабатывал (из-за включенных по умолчанию компонентов) и возвращал неверный
  текст ошибки.

#### HTTP

- Graceful shutdown не работал: нулевой `ShutdownTimeout` по умолчанию превращался в уже истекший
  контекст, и активные запросы обрывались. Теперь 0 означает «ограничение только контекстом
  точки входа», а по истечении таймаута соединения закрываются принудительно.
- У gateway таймаут остановки обнулялся внутри HTTP-сервера, у профайлера `ShutdownTimeout`
  не применялся вовсе.
- `WithTLSConfig` молча игнорировался: сервер отдавал незашифрованный HTTP.
- Обертка `ResponseWriter` в логирующем middleware скрывала `Flush` и `Hijack`: не работали
  server-streaming через gateway, SSE и WebSocket.
- Recovery-middleware перехватывал `http.ErrAbortHandler` и отвечал 500 вместо штатного
  прерывания ответа.
- Сервер с `nil`-хендлером отвечал 500 на любой запрос. Теперь это ошибка конфигурации.

#### gRPC

- Паника в стриминговом хендлере роняла весь процесс: для стримов не было ни одного интерцептора.
  Добавлены recovery, логирование и protovalidate.
- Gateway не закрывал соединение с gRPC-сервером при ошибке `Prepare` и при ошибке остановки
  HTTP-сервера.

#### Прочее

- `profiler.WithPort` принимал `*Config` вместо номера порта.
- `soma.WithLogger` не доходил до компонентов; логгер интерцепторов нельзя было заменить;
  `scheduler.WithLogger` не менял логгер gocron, а события задач писались в глобальный `slog`.
- `grpcserver.Prepare` и `scheduler.New` дописывали опции в переданный конфиг: повторное
  использование конфига дублировало интерцепторы и слушатели событий.

### Безопасность

- Профайлер (pprof) был включен всегда и слушал все интерфейсы: `/debug/pprof/cmdline` и дамп
  памяти были доступны из сети. Теперь он включается только `WithProfiler` и по умолчанию слушает
  `127.0.0.1`.
- HTTP-серверы, кроме профайлера, не имели таймаута чтения заголовков (уязвимость slowloris),
  и ни у одного не было таймаута простоя. Теперь по умолчанию `ReadHeaderTimeout` 10 с
  и `IdleTimeout` 2 мин.
- CORS (`*`), gRPC reflection и swagger были включены по умолчанию. Теперь все выключены
  и включаются явно: `WithCORSEnabled`, `WithReflectionEnabled`, `WithSwaggerEnabled`.
- `net/http/pprof` регистрирует хендлеры в `http.DefaultServeMux`. Теперь HTTP-сервер отказывается
  обслуживать `DefaultServeMux` (`httpserver.ErrDefaultServeMux`), чтобы pprof не утек наружу.

### Добавлено

- **Фаза Drain.** В начале остановки процессы, реализующие новый интерфейс `soma.Drainer`, перестают
  сообщать о готовности: gRPC health переходит в `NOT_SERVING`, `/healthz` gateway отвечает 503.
  Затем пауза `soma.WithShutdownDelay`, чтобы балансировщик (например, Kubernetes) успел снять
  экземпляр с трафика. Текущие запросы в это время продолжают обслуживаться.
- `soma.WithShutdownDelay`, `soma.WithCloseTimeout`.
- Интерфейс `soma.Named`: имя процесса в логах и ошибках. Реализован всеми компонентами.
- `soma.ErrGatewayRequiresGRPCServer`.
- Серверы занимают порт в `Prepare`: занятый порт обнаруживается до запуска остальных процессов.
- `Addr()` у gRPC-сервера, gateway, профайлера и HTTP-сервера — фактический адрес (полезно с портом 0).
- `grpcserver.Server.Drain()`.
- gRPC-интерцепторы для стримов: `LoggingAttrsStreamInterceptor`, `LoggingStreamInterceptor`,
  `RecoveryStreamInterceptor`.
- `grpcgateway`: `WithDialOptions` (в том числе TLS к gRPC-серверу), `WithHealthEndpointEnabled`,
  `WithHealthEndpointPath`, `WithSwaggerEnabled`, `WithSwaggerDir`, `WithSwaggerPath`,
  `WithCORSEnabled`, `WithCORSOptions`.
- `httpserver`: `WithName`, `WithHost`, `WithReadTimeout`, `WithWriteTimeout`, `WithIdleTimeout`,
  `WithMaxHeaderBytes`, `WithShutdownTimeout`, `Config.Validate`, `ErrNilHandler`, `ErrDefaultServeMux`.
- `profiler.WithHost`.
- Паника с ошибкой в функции закрытия ресурса оборачивается через `%w` и проверяется `errors.Is`.

### Изменено

- **Gateway подхватывает сервисы автоматически**: все сервисы gRPC-сервера, реализующие
  `grpcgateway.Service`. Пакеты `grpcserver` и `grpcgateway` больше не зависят друг от друга,
  связывает их `soma`.
- **Логгер наследуется**: компонент без собственного логгера использует логгер `soma.WithLogger`,
  без него — `slog.Default()`. Поле `Logger` в конфигах компонентов по умолчанию `nil`, атрибут
  `component` добавляется самим компонентом.
- **Встроенные интерцепторы и middleware включены всегда** и собираются в `Prepare`/`New` с актуальным
  логгером. Пользовательские опции (`WithServerOptions`, `WithMiddlewares`) выполняются внутри
  встроенных, поэтому их паники и коды ответа тоже попадают в лог.
- **Уровень лога запроса зависит от кода ответа**: успех — Info, ошибка клиента (4xx, `NotFound`,
  `InvalidArgument`, …) — Warn, ошибка сервиса (5xx, `Internal`, `Unavailable`, …) — Error.
  Сообщение одно для всех: `request handled`.
- `DefaultWith` сразу валидирует конфиг (`soma`, `httpserver`); `Default()` во всех пакетах
  больше не возвращает ошибку.
- Опции, принимающие списки, только дополняют их (`With...Options`, `WithMiddlewares`, `WithServices`).
- Контракт `Process` дополнен: `Shutdown` может быть вызван без `Run`, а при ошибке `Prepare`
  процесс сам освобождает захваченное.
- `httpserver.Server` больше не встраивает `*http.Server`; gateway и профайлер больше не встраивают
  `*httpserver.Server`, а явно делегируют ему `Run`, `Shutdown` и `Addr`.
- Каталоги пакетов переименованы без дефисов, чтобы совпадать с именами пакетов.
- Имена приведены к Go-конвенции для аббревиатур (`HTTP`, `GRPC`, `CORS`).

### Удалено

- `grpcserver.WithResetServerOptions`, `grpcserver.WithPrependServerOptions`,
  `grpcgateway.WithResetServeMuxOptions`, `httpserver.WithResetMiddleware`,
  `scheduler.WithResetSchedulerOptions`, `scheduler.WithResetGlobalJobOptions`: встроенное
  больше не лежит в этих списках, а значения по умолчанию переопределяются добавлением
  (для gocron и grpc-gateway последняя опция побеждает).
- `profiler.WithReadHeaderTimeout`: профайлер использует значения по умолчанию HTTP-сервера.

### Инфраструктура

- Тесты на все компоненты и жизненный цикл (24 тестовые функции, table-driven, testify + gomock).
  Ключевые исправления проверены мутациями: если вернуть старое поведение, падает хотя бы один тест.
- Моки (`go.uber.org/mock`) для `Process`, `Named`, `Drainer`, `Closer`, `grpcserver.Service`,
  `grpcgateway.Service`; генерация — `task gen:mock`.
- Конфигурация golangci-lint v2 (`.golangci.yml`): стандартный набор, bodyclose, errorlint, gosec,
  noctx и проверки именования staticcheck.
- Задачи Taskfile: `tool:install` (mockgen и golangci-lint в `./bin`), `test`, `lint`, `lint:fix`.

### Миграция с v1.1.14

#### Импорты

| Было | Стало |
|------|-------|
| `github.com/murouse/soma/component/grpc-server` | `github.com/murouse/soma/component/grpcserver` |
| `github.com/murouse/soma/component/grpc-gateway` | `github.com/murouse/soma/component/grpcgateway` |
| `github.com/murouse/soma/component/http-server` | `github.com/murouse/soma/component/httpserver` |
| `github.com/murouse/soma/accessor/middleware/grpc-interceptor` | `github.com/murouse/soma/accessor/middleware/grpcinterceptor` |
| `github.com/murouse/soma/accessor/middleware/http-middleware` | `github.com/murouse/soma/accessor/middleware/httpmiddleware` |

#### API

| Было | Стало |
|------|-------|
| `soma.WithGrpcServer` | `soma.WithGRPCServer` |
| `soma.WithGrpcGateway` | `soma.WithGRPCGateway` |
| `soma.WithHttpServer` | `soma.WithHTTPServer` |
| `soma.Default() (*EntrypointConfig, error)` | `soma.Default() *EntrypointConfig` |
| `grpcserver.ImplementationAdapter` | `grpcserver.Service` (только `RegisterServer`) + `grpcgateway.Service` (только `RegisterHandler`) |
| `grpcserver.WithAdapters` | `grpcserver.WithServices` |
| `grpcserver.WithAppendServerOptions` | `grpcserver.WithServerOptions` |
| `grpcserver.WithReflection` | `grpcserver.WithReflectionEnabled` |
| `grpcserver.WithHealthServer` | `grpcserver.WithHealthServerEnabled` |
| `grpcserver.Default() (*Config, error)` | `grpcserver.Default() *Config` |
| `grpcserver.Config.Impls` | `grpcserver.Config.Services` |
| `grpcgateway.New(cfg, impls, grpcPort)` | `grpcgateway.New(cfg, services, grpcTarget)`, например `"localhost:1482"` |
| `grpcgateway.WithHttpHandler` | `grpcgateway.WithHTTPHandler` |
| `grpcgateway.WithCors(opts)` | `grpcgateway.WithCORSEnabled(true)` + `grpcgateway.WithCORSOptions(opts)` |
| `grpcgateway.WithSwagger(dir, path)` | `grpcgateway.WithSwaggerEnabled(true)` + `WithSwaggerDir(dir)` + `WithSwaggerPath(path)` |
| `grpcgateway.WithHealthEndpoint(&path)` / `(nil)` | `grpcgateway.WithHealthEndpointPath(path)` / `WithHealthEndpointEnabled(false)` |
| `httpserver.WithMiddleware` | `httpserver.WithMiddlewares` |
| `profiler.WithPort(&profiler.Config{Port: p})` | `profiler.WithPort(p)` |
| `middleware.LogRequestFinal(ctx, logger, start, isError, …)` | `middleware.LogRequestFinal(ctx, logger, level, start, …)` |

#### Поведение

- Сервису, которому нужен компонент, теперь нужна его опция `With*`: без нее компонент не запускается.
- Чтобы сервис был доступен через gateway, он должен реализовывать `grpcgateway.Service`
  (метод `RegisterHandler`) и быть передан в `grpcserver.WithServices`.
- CORS, swagger и reflection нужно включить явно, если они используются.
- Профайлер слушает `127.0.0.1`. Для доступа из сети (например, в Docker с пробросом порта) —
  `profiler.WithHost("")`.
- `soma.WithHTTPServer(nil, …)` и `soma.WithHTTPServer(http.DefaultServeMux, …)` теперь возвращают ошибку.
- Если в Kubernetes нужна остановка без ошибок у клиентов, задайте `soma.WithShutdownDelay`
  (например, 5 с) и `terminationGracePeriodSeconds` больше суммы
  `shutdownDelay + shutdownTimeout + closeTimeout`.
