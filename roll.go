package main

import (
	"fmt"
	"math/rand"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer  = otel.Tracer("roll")
	meter   = otel.Meter("roll")
	rollCnt metric.Int64Counter
)

func init() {
	var err error
	rollCnt, err = meter.Int64Counter("dice.rolls",
		metric.WithDescription("The number of rolls by roll value"),
		metric.WithUnit("{roll}"))
	if err != nil {
		panic(err)
	}
}

func roll(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "roll") // 开始 span
	defer span.End()                               // 结束 span

	number := 1 + rand.Intn(6)

	rollValueAttr := attribute.Int("roll.value", number)

	span.SetAttributes(rollValueAttr) // span 添加属性

	// 摇骰子次数的指标 +1
	rollCnt.Add(ctx, 1, metric.WithAttributes(rollValueAttr))

	_, _ = fmt.Fprintln(w, number)
}
