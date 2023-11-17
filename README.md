# Decorender

A library for declarative rendering on the backend. Considering that there is no goal to replicate browser rendering, a simple positioning model has been implemented.


## Installation

    go get -u github.com/godknowsiamgood/decorender

## Usage

```go
// First, we create a renderer object that reads the yaml file 
// and initializes the necessary resources
renderer, err := decorender.NewRenderer("./layout.yaml")

// Then it can be used multiple times 
// with different data and is concurrent-safe.
renderer.Render(nil)
renderer.Render(yourData)
renderer.RenderToFile(yourData, "result.jpg")
```

## Format

See `test.yaml` and `test.png`. More detailed documentation, I think, will appear later.

## Performance

At the moment, no special tuning of performance and optimization of memory consumption has been carried out.
