# server-core 

This is package to create workers pull with graceful shutdown and panics catcher.

### Tu use it import package:

```go get github.com/vkidmode/server-core/pkg/core```

### Create new server item with: 

```app := core.NewCore(customLogger, timeout, workersCount)```

* you can use custom logger to log panic stack
* timeout used for shutting down app if it cant stop
* workers count is maximum count of simultaneously running workers

### Add worker to app:

```app.AddRunner(runner)```

* It is possible to add runners in runtime

### Launch app:

```err = app.Launch(ctx)```

* app will finish when all workers finish or error in worker happen


