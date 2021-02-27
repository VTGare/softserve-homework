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
