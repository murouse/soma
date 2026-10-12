# Soma

**Soma** — это декларативный фреймворк для быстрой сборки микросервисов на Go. Он предоставляет единый жизненный цикл для gRPC-серверов, HTTP-шлюзов, профайлеров и фоновых задач.

> **In biological terms, Soma is the cell body where services live, execute, and are maintained.**
>
> **It acts as the operational center of each runtime unit, responsible for sustaining execution, coordinating state, and integrating system-level signals such as communication, observability, and lifecycle events.**

## Содержание

- [Основные возможности](#основные-возможности)
- [Установка](#установка)
- [Быстрый старт](#быстрый-старт)
- [Полный пример](#полный-пример)
- [Жизненный цикл](#жизненный-цикл)
- [Справочник опций](#справочник-опций)
- [Свои процессы](#свои-процессы)
- [gRPC-сервисы и gateway](#grpc-сервисы-и-gateway)
- [Логирование](#логирование)
- [Безопасность по умолчанию](#безопасность-по-умолчанию)
- [Kubernetes](#kubernetes)
- [Разработка](#разработка)

## Основные возможности

* **Декларативная конфигурация**: компоненты подключаются и настраиваются через `Functional Options`.
* **Единый жизненный цикл**: подготовка, запуск, плавная остановка и закрытие ресурсов для всех процессов.
* **Готовность к Kubernetes**: health-проверки, снятие с балансировки перед остановкой, ограниченное время остановки.
* **Модульность**: gRPC-сервер, HTTP-шлюз (grpc-gateway), HTTP-серверы, планировщик задач и pprof — каждый включается своей опцией.
* **Безопасные значения по умолчанию**: CORS, reflection и swagger выключены, профайлер доступен только локально, у HTTP-серверов есть таймауты.
* **Наблюдаемость из коробки**: логирование запросов, recovery от паник и валидация (protovalidate) для HTTP и gRPC, включая стримы.
* **Type-safe**: Полная поддержка типизации благодаря Go.

## Установка

```bash
go get github.com/murouse/soma
```

## Быстрый старт

```go
func main() {
    err := soma.Run(context.Background(),
        soma.WithGRPCServer(
            grpcserver.WithServices(&UserService{}),
        ),
        soma.WithGRPCGateway(),
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

Этого достаточно, чтобы получить gRPC-сервер на `:1482` с health-сервером и HTTP-шлюз на `:1480` с `/healthz`.
Сервис останавливается по SIGINT/SIGTERM: дожидается активных запросов и закрывает соединения.

## Полный пример

Пример использует все опции всех компонентов. Ни одна из них не обязательна: всё, что не задано,
берется из значений по умолчанию (см. [справочник](#справочник-опций)). `userv1` — код, сгенерированный
`protoc-gen-go-grpc` и `protoc-gen-grpc-gateway`.

```go
package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/go-chi/cors"
	"github.com/go-co-op/gocron/v2"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/murouse/soma"
	"github.com/murouse/soma/component/grpcgateway"
	"github.com/murouse/soma/component/grpcserver"
	"github.com/murouse/soma/component/httpserver"
	"github.com/murouse/soma/component/profiler"
	"github.com/murouse/soma/component/scheduler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	userv1 "example.com/app/gen/user/v1"
)

// UserService — gRPC-сервис. RegisterServer делает его grpcserver.Service,
// а RegisterHandler — grpcgateway.Service: gateway подхватит его автоматически.
type UserService struct {
	userv1.UnimplementedUserServiceServer
}

func (s *UserService) RegisterServer(server *grpc.Server) {
	userv1.RegisterUserServiceServer(server, s)
}

func (s *UserService) RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return userv1.RegisterUserServiceHandler(ctx, mux, conn)
}

// Consumer — собственный фоновый процесс: реализует soma.Process,
// а также опциональные soma.Named и soma.Drainer.
type Consumer struct {
	paused atomic.Bool
	stop   chan struct{}
}

func NewConsumer() *Consumer {
	return &Consumer{stop: make(chan struct{})}
}

// Name — имя процесса в логах и ошибках (soma.Named).
func (c *Consumer) Name() string { return "consumer" }

// Prepare — подключение к брокеру. При ошибке Prepare должен сам освободить то, что успел захватить.
func (c *Consumer) Prepare(ctx context.Context) error { return nil }

// Run блокируется, пока не отменится ctx или не будет вызван Shutdown.
func (c *Consumer) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.stop:
			return nil
		case <-ticker.C:
			if !c.paused.Load() {
				// забрать и обработать очередную порцию сообщений
			}
		}
	}
}

// Drain — начало остановки (soma.Drainer): перестаем брать новые сообщения.
func (c *Consumer) Drain() { c.paused.Store(true) }

// Shutdown разблокирует Run и освобождает ресурсы процесса.
func (c *Consumer) Shutdown(ctx context.Context) error {
	close(c.stop)
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	debug := os.Getenv("DEBUG") == "true"

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		logger.Error("open db", "error", err)
		os.Exit(1)
	}

	cert, err := tls.LoadX509KeyPair("tls/cert.pem", "tls/key.pem")
	if err != nil {
		logger.Error("load certificate", "error", err)
		os.Exit(1)
	}

	version := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("1.0.0"))
	})

	webhooks := http.NewServeMux()
	webhooks.HandleFunc("POST /payments", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	requireToken := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Webhook-Token") != os.Getenv("WEBHOOK_TOKEN") {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	err = soma.Run(context.Background(),
		// Точка входа
		soma.WithLogger(logger),
		soma.WithPrepareTimeout(10*time.Second),
		soma.WithShutdownDelay(5*time.Second),
		soma.WithShutdownTimeout(15*time.Second),
		soma.WithCloseTimeout(5*time.Second),
		soma.WithClosers(db),
		soma.WithProcesses(NewConsumer()),

		// gRPC-сервер
		soma.WithGRPCServer(
			grpcserver.WithPort(1482),
			grpcserver.WithServices(&UserService{}),
			grpcserver.WithReflectionEnabled(debug),
			grpcserver.WithHealthServerEnabled(true),
			grpcserver.WithUnaryTimeout(30*time.Second),
			grpcserver.WithServerOptions(
				grpc.MaxRecvMsgSize(16<<20),
				grpc.KeepaliveParams(keepalive.ServerParameters{MaxConnectionIdle: 5 * time.Minute}),
			),
			grpcserver.WithLogger(logger.With("api", "public")),
		),

		// HTTP-шлюз к gRPC-серверу
		soma.WithGRPCGateway(
			grpcgateway.WithPort(1480),
			grpcgateway.WithShutdownTimeout(5*time.Second),
			grpcgateway.WithServeMuxOptions(
				runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
					if key == "X-Request-Id" {
						return "x-request-id", true
					}
					return runtime.DefaultHeaderMatcher(key)
				}),
			),
			grpcgateway.WithDialOptions(
				grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16<<20)),
			),
			grpcgateway.WithHTTPHandler("/version", version),
			grpcgateway.WithHealthEndpointEnabled(true),
			grpcgateway.WithHealthEndpointPath("/healthz"),
			grpcgateway.WithSwaggerEnabled(debug),
			grpcgateway.WithSwaggerDir("./docs/swagger"),
			grpcgateway.WithSwaggerPath("/swagger/"),
			grpcgateway.WithCORSEnabled(true),
			grpcgateway.WithCORSOptions(cors.Options{
				AllowedOrigins:   []string{"https://app.example.com"},
				AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete},
				AllowedHeaders:   []string{"Authorization", "Content-Type"},
				AllowCredentials: true,
				MaxAge:           300,
			}),
			grpcgateway.WithLogger(logger),
		),

		// Профайлер pprof
		soma.WithProfiler(
			profiler.WithHost("127.0.0.1"),
			profiler.WithPort(1491),
			profiler.WithShutdownTimeout(time.Second),
			profiler.WithLogger(logger),
		),

		// Планировщик задач
		soma.WithScheduler(
			scheduler.WithJobs(scheduler.Job{
				Definition: gocron.DurationJob(time.Hour),
				Task: gocron.NewTask(func(ctx context.Context) error {
					_, err := db.ExecContext(ctx, "DELETE FROM sessions WHERE expires_at < now()")
					return err
				}),
				Options: []gocron.JobOption{gocron.WithName("cleanup-sessions")},
			}),
			scheduler.WithSchedulerOptions(gocron.WithLocation(time.UTC)),
			scheduler.WithGlobalJobOptions(gocron.WithSingletonMode(gocron.LimitModeReschedule)),
			scheduler.WithLoggingJobEnabled(true),
			scheduler.WithLogger(logger),
		),

		// Дополнительный HTTP-сервер (их может быть сколько угодно)
		soma.WithHTTPServer(webhooks, 8443,
			httpserver.WithName("webhooks"),
			httpserver.WithHost(""),
			httpserver.WithTLSConfig(&tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}),
			httpserver.WithReadTimeout(10*time.Second),
			httpserver.WithReadHeaderTimeout(5*time.Second),
			httpserver.WithWriteTimeout(30*time.Second),
			httpserver.WithIdleTimeout(2*time.Minute),
			httpserver.WithMaxHeaderBytes(1<<20),
			httpserver.WithShutdownTimeout(10*time.Second),
			httpserver.WithMiddlewares(requireToken),
			httpserver.WithLogger(logger),
		),
	)
	if err != nil {
		logger.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}
```

Что здесь происходит:

- `debug` управляет reflection и swagger: код одинаков для всех окружений, различается только конфиг.
- `WithLogger` у компонентов указан для полноты: без него компонент наследует логгер `soma.WithLogger`.
  У gRPC-сервера логгер переопределен, чтобы к его логам добавлялся атрибут `api`.
- Ошибка задачи планировщика попадает в лог с уровнем Error, а `ctx` задачи отменяется при остановке.
- `db` закрывается последним — после того как остановлены все серверы, планировщик и `Consumer`.

## Жизненный цикл

`soma.Run` (или `NewEntrypoint` + `Entrypoint.Run`) проводит все процессы через пять фаз:

1. **Prepare** — последовательно, в порядке процессов, с общим таймаутом `WithPrepareTimeout`.
   Серверы на этом шаге занимают порты, поэтому занятый порт обнаруживается до запуска остальных.
   Если `Prepare` одного процесса падает, уже подготовленные останавливаются в обратном порядке,
   а ресурсы закрываются.
2. **Run** — все процессы конкурентно. Фаза длится, пока не придет SIGINT/SIGTERM, не отменится
   переданный контекст, один из процессов не вернет ошибку или все процессы не завершатся сами.
3. **Drain** — процессы, реализующие `soma.Drainer`, перестают сообщать о готовности: gRPC health
   переходит в `NOT_SERVING`, `/healthz` gateway отвечает 503. Затем пауза `WithShutdownDelay`.
   Текущие и новые запросы в это время продолжают обслуживаться.
4. **Shutdown** — процессы останавливаются в обратном порядке, затем ожидается завершение их `Run`.
   На всё вместе отводится `WithShutdownTimeout`. Серверы дожидаются активных запросов, а по истечении
   времени закрывают соединения принудительно.
5. **Close** — ресурсы из `WithClosers` закрываются в обратном порядке с отдельным бюджетом
   `WithCloseTimeout`: даже если процессы израсходовали свое время, ресурсы получают шанс закрыться.

```
SIGTERM
  │
  ├─ Drain ──────────────────────── health: NOT_SERVING
  ├─ пауза shutdownDelay ─────────── запросы обслуживаются
  ├─ Shutdown (LIFO) + ожидание Run ─ не дольше shutdownTimeout
  └─ Close (LIFO) ────────────────── не дольше closeTimeout
```

Максимальное время остановки: `shutdownDelay + shutdownTimeout + closeTimeout`.

**Порядок процессов.** Процессы из `WithProcesses` → планировщик → gRPC-сервер → gateway → профайлер →
HTTP-серверы (в порядке `WithHTTPServer`). Prepare идет в этом порядке, Drain и Shutdown — в обратном:
например, gateway останавливается раньше gRPC-сервера, в который он проксирует запросы.

**Ошибки.** `Run` возвращает объединение (`errors.Join`) ошибок всех фаз, в каждой указано имя процесса.
Ошибка `context.Canceled` из `Run` процесса не считается ошибкой. Процесс, вернувший `nil`,
не останавливает остальные.

**Сигналы.** Подписка на SIGINT/SIGTERM действует с самого начала, включая Prepare. В начале остановки
она снимается: повторный Ctrl+C во время зависшей остановки завершает процесс сразу.

## Справочник опций

### Точка входа (`soma`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithLogger(*slog.Logger)` | `slog.Default()` | Логгер точки входа. Компоненты без собственного логгера наследуют его. |
| `WithProcesses(...Process)` | — | Свои процессы. Запускаются первыми, останавливаются последними. |
| `WithPrepareTimeout(time.Duration)` | 5 с | Общий таймаут фазы Prepare. |
| `WithShutdownDelay(time.Duration)` | 0 | Пауза между Drain и Shutdown. |
| `WithShutdownTimeout(time.Duration)` | 5 с | Shutdown всех процессов и ожидание их `Run`. |
| `WithCloseTimeout(time.Duration)` | 5 с | Закрытие ресурсов из `WithClosers`. |
| `WithClosers(...io.Closer)` | — | Ресурсы (БД, брокеры), закрываемые после остановки процессов в обратном порядке. |
| `WithScheduler(...scheduler.Option)` | выключен | Планировщик задач. |
| `WithGRPCServer(...grpcserver.Option)` | выключен | gRPC-сервер. |
| `WithGRPCGateway(...grpcgateway.Option)` | выключен | HTTP-шлюз. Требует `WithGRPCServer`, иначе `ErrGatewayRequiresGRPCServer`. |
| `WithProfiler(...profiler.Option)` | выключен | Профайлер pprof. |
| `WithHTTPServer(http.Handler, int, ...httpserver.Option)` | — | HTTP-сервер с хендлером и портом. Можно добавить несколько. |

### gRPC-сервер (`grpcserver`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithPort(int)` | 1482 | Порт; слушает все интерфейсы. |
| `WithServices(...Service)` | — | Сервисы. Те, что реализуют `grpcgateway.Service`, попадают и в gateway. |
| `WithReflectionEnabled(bool)` | `false` | gRPC reflection (для grpcurl, Postman). |
| `WithHealthServerEnabled(bool)` | `true` | `grpc.health.v1.Health`; переходит в `NOT_SERVING` на фазе Drain. |
| `WithUnaryTimeout(time.Duration)` | 1 мин | Дедлайн для unary-запросов без клиентского дедлайна; 0 — без него. |
| `WithServerOptions(...grpc.ServerOption)` | — | Опции `grpc.Server` (TLS через `grpc.Creds`, лимиты, keepalive). Интерцепторы отсюда выполняются внутри встроенных. |
| `WithLogger(*slog.Logger)` | наследуется | Логгер сервера и интерцепторов. |

Встроенные интерцепторы:

- unary: атрибуты логов → лог запроса → recovery → protovalidate → таймаут;
- stream: атрибуты логов → лог запроса → recovery → protovalidate.

### HTTP-шлюз (`grpcgateway`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithPort(int)` | 1480 | Порт; слушает все интерфейсы. |
| `WithShutdownTimeout(time.Duration)` | 1 с | Ожидание активных запросов при остановке. |
| `WithServeMuxOptions(...runtime.ServeMuxOption)` | JSON: `EmitUnpopulated`, имена полей в camelCase | Опции grpc-gateway. Добавляются к встроенным; для маршалера побеждает последний. |
| `WithDialOptions(...grpc.DialOption)` | без TLS | Опции соединения с gRPC-сервером. Применяются после встроенных, поэтому `grpc.WithTransportCredentials` включает TLS. |
| `WithHTTPHandler(string, http.Handler)` | — | Свой хендлер на паттерн `http.ServeMux`. Имеет приоритет над grpc-gateway. |
| `WithHealthEndpointEnabled(bool)` | `true` | HTTP-эндпоинт, проксирующий gRPC health: 200 или 503. |
| `WithHealthEndpointPath(string)` | `/healthz` | Путь health-эндпоинта. |
| `WithSwaggerEnabled(bool)` | `false` | Раздача спецификации OpenAPI с диска. |
| `WithSwaggerDir(string)` | `./docs/swagger` | Каталог со спецификацией. |
| `WithSwaggerPath(string)` | `/swagger/` | HTTP-путь спецификации. |
| `WithCORSEnabled(bool)` | `false` | CORS. |
| `WithCORSOptions(cors.Options)` | любые origin и заголовки, методы HEAD/GET/POST/PUT/PATCH/DELETE, без credentials | Настройки CORS (`github.com/go-chi/cors`). Действуют, только если CORS включен. |
| `WithLogger(*slog.Logger)` | наследуется | Логгер шлюза. |

Маршрутизация: сначала хендлеры `WithHTTPHandler` и swagger, всё остальное — в grpc-gateway.
HTTP-сервер шлюза использует значения по умолчанию `httpserver` (таймауты, встроенные middleware).
Шлюз подключается к gRPC-серверу по `localhost:<порт gRPC-сервера>`.

### HTTP-сервер (`httpserver`, через `soma.WithHTTPServer`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithName(string)` | `http-server` | Имя в логах (атрибут `component`) и ошибках. |
| `WithHost(string)` | `""` (все интерфейсы) | Интерфейс для прослушивания. |
| `WithTLSConfig(*tls.Config)` | без TLS | Включает HTTPS. Сертификаты — в `Certificates` или `GetCertificate`. |
| `WithReadTimeout(time.Duration)` | 0 | `http.Server.ReadTimeout`. |
| `WithReadHeaderTimeout(time.Duration)` | 10 с | Защита от slowloris. |
| `WithWriteTimeout(time.Duration)` | 0 | `http.Server.WriteTimeout`. |
| `WithIdleTimeout(time.Duration)` | 2 мин | Время жизни простаивающих keep-alive соединений. |
| `WithMaxHeaderBytes(int)` | 0 (1 МБ по умолчанию `net/http`) | Максимальный размер заголовков. |
| `WithShutdownTimeout(time.Duration)` | 0 | Ожидание активных запросов; 0 — ограничение только `WithShutdownTimeout` точки входа. |
| `WithMiddlewares(...func(http.Handler) http.Handler)` | — | Свои middleware; первый — внешний. Выполняются внутри встроенных. |
| `WithLogger(*slog.Logger)` | наследуется | Логгер сервера и middleware. |

Встроенные middleware: атрибуты логов → лог запроса → recovery → ваши middleware → хендлер.
Хендлер `nil` и `http.DefaultServeMux` отклоняются (`ErrNilHandler`, `ErrDefaultServeMux`).

### Профайлер (`profiler`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithHost(string)` | `127.0.0.1` | Интерфейс. `""` — все интерфейсы: открывайте pprof наружу осознанно. |
| `WithPort(int)` | 1491 | Порт. |
| `WithShutdownTimeout(time.Duration)` | 1 с | Ожидание активных запросов при остановке. |
| `WithLogger(*slog.Logger)` | наследуется | Логгер профайлера. |

Эндпоинты: `/debug/pprof/` (индекс и `heap`), `goroutine`, `allocs`, `block`, `mutex`, `threadcreate`,
`profile`, `trace`, `symbol`, `cmdline`. Профили `block` и `mutex` пусты, пока не включены
`runtime.SetBlockProfileRate` и `runtime.SetMutexProfileFraction`.

### Планировщик (`scheduler`)

| Опция | По умолчанию | Описание |
|-------|--------------|----------|
| `WithJobs(...Job)` | — | Задачи: `Job{Definition, Task, Options}` из `gocron/v2`. |
| `WithSchedulerOptions(...gocron.SchedulerOption)` | `gocron.WithLocation(time.Local)` | Опции планировщика; побеждает последняя. |
| `WithGlobalJobOptions(...gocron.JobOption)` | — | Опции для всех задач. Используйте вместо `gocron.WithGlobalJobOptions`: та перезаписывается встроенными. |
| `WithLoggingJobEnabled(bool)` | `true` | Логировать старт и завершение задач (Debug) и ошибки (Error). |
| `WithLogger(*slog.Logger)` | наследуется | Логгер планировщика и gocron. |

Если первый параметр функции задачи — `context.Context`, gocron передаст в него контекст задачи
и отменит его при остановке. Возвращенная ошибка попадает в лог.

## Свои процессы

Любой тип, реализующий `soma.Process`, подключается через `soma.WithProcesses`:

```go
type Process interface {
	Prepare(ctx context.Context) error  // инициализация; при ошибке процесс сам освобождает захваченное
	Run(ctx context.Context) error      // блокируется до отмены ctx или вызова Shutdown
	Shutdown(ctx context.Context) error // разблокирует Run и освобождает ресурсы, уважая ctx
}
```

Контракт:

- `Run` обязан завершиться при отмене `ctx` или после `Shutdown`. Ошибка, отличная от
  `context.Canceled`, останавливает все приложение; `nil` не останавливает остальные процессы.
- `Shutdown` может быть вызван после `Prepare`, но без `Run` (при ошибке Prepare другого процесса
  или раннем сигнале), а `Run` после `Shutdown` должен сразу завершаться.
- Если `Prepare` вернул ошибку, `Shutdown` для этого процесса не вызывается.

Опциональные интерфейсы:

| Интерфейс | Метод | Зачем |
|-----------|-------|-------|
| `soma.Named` | `Name() string` | Имя процесса в логах и ошибках вместо имени типа. |
| `soma.Drainer` | `Drain()` | Реакция на начало остановки (до паузы и Shutdown). Должен быть быстрым и неблокирующим. |

Пример — `Consumer` в [полном примере](#полный-пример).

## gRPC-сервисы и gateway

Сервис регистрируется на gRPC-сервере через `grpcserver.Service`. Если он также реализует
`grpcgateway.Service`, gateway подхватывает его автоматически — передавать сервис второй раз не нужно:

```go
// grpcserver.Service
type Service interface {
	RegisterServer(server *grpc.Server)
}

// grpcgateway.Service
type Service interface {
	RegisterHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error
}
```

Сервис без `RegisterHandler` доступен только по gRPC. Если gRPC-сервер работает с TLS
(`grpcserver.WithServerOptions(grpc.Creds(...))`), передайте gateway креденшелы клиента:
`grpcgateway.WithDialOptions(grpc.WithTransportCredentials(...))`.

## Логирование

- Компонент без собственного логгера наследует логгер `soma.WithLogger`, а без него — `slog.Default()`.
  Каждый компонент добавляет атрибут `component` (`entrypoint`, `grpc-server`, `grpc-gateway`,
  `profiler`, `scheduler`, `closer` или имя HTTP-сервера).
- Каждый запрос логируется сообщением `request handled` с длительностью и кодом ответа. Уровень зависит
  от кода: успех — Info, ошибка клиента (4xx, `NotFound`, `InvalidArgument`, `Unauthenticated`, …) — Warn,
  ошибка сервиса (5xx, `Internal`, `Unavailable`, `DeadlineExceeded`, …) — Error.
- Health-проверки (`/healthz`, `/readyz`, `grpc.health.v1.Health`) не логируются.
- Атрибуты запроса (метод, путь или gRPC-метод, адрес клиента, `User-Agent`) кладутся в контекст через
  `golgi.WithAttrs` и попадают во все логи хендлера, если логгер построен через `golgi`.
- Паники перехватываются: в лог пишутся значение паники и стек, клиент получает 500 или `codes.Internal`.

## Безопасность по умолчанию

- CORS, gRPC reflection и swagger выключены; включаются опциями `With...Enabled(bool)`.
- Профайлер включается только `soma.WithProfiler` и слушает `127.0.0.1`.
- `net/http/pprof` при импорте регистрирует `/debug/pprof` в `http.DefaultServeMux`, поэтому
  HTTP-серверы soma отказываются его обслуживать. В собственном коде тоже не отдавайте
  `DefaultServeMux` наружу.
- У HTTP-серверов есть `ReadHeaderTimeout` и `IdleTimeout`.
- У unary-запросов без клиентского дедлайна есть серверный дедлайн (`WithUnaryTimeout`).

## Kubernetes

- **Readiness** — `/healthz` gateway (`httpGet`) или gRPC health (`grpc: {port: 1482}`). На фазе Drain
  обе проверки сообщают о неготовности.
- **Остановка без ошибок у клиентов** — `soma.WithShutdownDelay(5*time.Second)`: за это время Kubernetes
  убирает под из балансировки, а запросы продолжают обслуживаться.
- **`terminationGracePeriodSeconds`** должен быть больше `shutdownDelay + shutdownTimeout + closeTimeout`,
  иначе Kubernetes завершит процесс по SIGKILL до конца остановки.
- **Профайлер** доступен через `kubectl port-forward pod/<pod> 1491`, наружу он не открыт.

## Разработка

```bash
task tool:install # mockgen и golangci-lint в ./bin
task test         # тесты с race-детектором
task lint         # golangci-lint
task lint:fix     # golangci-lint с автоисправлениями
task gen:mock     # перегенерация моков
```

Серверы с портом 0 занимают свободный порт; фактический адрес после `Prepare` возвращает `Addr()` —
это удобно в интеграционных тестах.

Изменения между версиями — в [CHANGELOG.md](CHANGELOG.md).
