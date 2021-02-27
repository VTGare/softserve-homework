# softserve-homework

softserve-homework is a coding assignment for my Softserve job application.  
A simple post microservice powered by Redis.

## Requirements
- Go (1.16+)
- Redis (6.0+)

## Launching
softserve-homework requires a **config.json** file in **working directory** to work.

**Example configuration:**
```
{
    "redis": {
        "host": "127.0.0.1",
        "port": "6379"
    },
    "host": "",
    "port": "3000"
}
```

## Project layout
1. `cmd/post` - project's main application and entry point.
2. `internal` - private application code used around all packages.
    - `config` - app configuration package.
    - `database` - creates database connection.
    - `middlewares` - collection of useful net/http compatible middlewares.
3. `pkg/post` - application's business logic.