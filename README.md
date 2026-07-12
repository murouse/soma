# Soma

**Soma** — это декларативный фреймворк для быстрой сборки микросервисов на Go. Он предоставляет единый жизненный цикл для gRPC-серверов, HTTP-шлюзов, профайлеров и фоновых задач.

> **In biological terms, Soma is the cell body where services live, execute, and are maintained.**
>
> **It acts as the operational center of each runtime unit, responsible for sustaining execution, coordinating state, and integrating system-level signals such as communication, observability, and lifecycle events.**

## Основные возможности

* **Декларативная конфигурация**: Настройка компонентов через систему `Functional Options`.
* **Единый жизненный цикл**: Автоматическая обработка `Graceful Shutdown` для всех процессов.
* **Модульность**: Легкое подключение gRPC, HTTP Gateway, Scheduler и pprof.
* **Type-safe**: Полная поддержка типизации благодаря Go.

## Пример использования

```go
func main() {
    ctx := context.Background()

    err := soma.NewEntrypointRun(ctx,
        soma.WithGrpcServer(
            grpcserver.WithAdapters(myService),
            grpcserver.WithPort(9090),
        ),
        soma.WithGrpcGateway(
            grpcgateway.WithPort(8080),
        ),
        soma.WithProfiler(),
    )
    if err != nil {
        log.Fatal(err)
    }
}

```

## Установка

```bash
go get github.com/murouse/soma

```

## Как это работает?

Фреймворк выступает в роли "ядра клетки", оркестрируя запуск и остановку всех подсистем вашего сервиса. Он следит за тем, чтобы все компоненты (gRPC, Gateway, Scheduler) стартовали в правильном порядке и корректно освобождали ресурсы при получении системных сигналов.
