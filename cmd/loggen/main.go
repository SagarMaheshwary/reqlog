package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

var format = flag.String("format", "json", "log output format: json or text")
var orders = flag.Int("orders", 100, "number of 'order created' events to generate (produces multiple logs per service per event)")

func main() {
	flag.Parse()

	r := rand.New(rand.NewSource(42))

	os.MkdirAll("./logs", 0o755)

	orderFile, _ := os.Create(filepath.Join("./logs", "order-service.log"))
	defer orderFile.Close()

	inventoryFile, _ := os.Create(filepath.Join("./logs", "inventory-service.log"))
	defer inventoryFile.Close()

	base := time.Date(2026, 3, 22, 15, 0, 0, 0, time.UTC)

	offsetOrder := 1
	offsetInventory := 1

	hot := map[int]bool{72: true, 105: true, 999: true, 2048: true}

	noiseMsgs := []string{
		"health check ok", "worker heartbeat", "retrying downstream call",
		"cache miss", "publishing metric", "consumer idle",
		"db query completed", "http request completed",
	}

	for i := 1; i <= *orders; i++ {
		ts := base.Add(time.Duration(i*50+r.Intn(200)) * time.Millisecond)
		rid := reqID(r, i)

		// noise (no request_id)
		for j := 0; j < r.Intn(3); j++ {
			logLine(orderFile, ts, "INF", noiseMsgs[r.Intn(len(noiseMsgs))],
				fmt.Sprintf("latency_ms=%d instance=order-%d", 5+r.Intn(900), 1+r.Intn(3)),
				0, "")
		}
		for j := 0; j < r.Intn(2); j++ {
			logLine(inventoryFile, ts, "INF", noiseMsgs[r.Intn(len(noiseMsgs))],
				fmt.Sprintf("latency_ms=%d instance=inventory-%d", 5+r.Intn(900), 1+r.Intn(3)),
				0, "")
		}

		items := randomItems(r)

		// order.created
		val := fmt.Sprintf(`{"items":%s,"order_id":%d}`, items, i)

		if *format == "text" {
			logLine(orderFile, ts, "INF", "Message received",
				fmt.Sprintf(`event_key=order.created offset=%d topic=outbox.events value=%q`, offsetOrder, val),
				i, rid)
		} else {
			logLine(orderFile, ts, "INF", "Message received",
				fmt.Sprintf(`,"event_key":"order.created","offset":%d,"topic":"outbox.events","value":%q`, offsetOrder, val),
				i, rid)
		}
		offsetOrder++

		if r.Float64() < 0.65 {
			logLine(orderFile, ts, "INF", "Ignoring event",
				"event=order.created", 0, rid)
		}

		if r.Float64() < 0.82 || hot[i] {
			if *format == "text" {
				logLine(inventoryFile, ts, "INF", "Message received",
					fmt.Sprintf(`event_key=order.created offset=%d topic=outbox.events value=%q`, offsetInventory, val),
					i, rid)
			} else {
				logLine(inventoryFile, ts, "INF", "Message received",
					fmt.Sprintf(`,"event_key":"order.created","offset":%d,"topic":"outbox.events","value":%q`, offsetInventory, val),
					i, rid)
			}
			offsetInventory++
		}

		// inventory.reserved
		if r.Float64() < 0.35 || hot[i] {
			reserveTS := ts.Add(time.Duration(500+r.Intn(5000)) * time.Millisecond)
			val2 := fmt.Sprintf(`{"order_id":%d}`, i)

			if *format == "text" {
				logLine(orderFile, reserveTS, "INF", "Message received",
					fmt.Sprintf(`event_key=inventory.reserved offset=%d topic=outbox.events value=%q`, offsetOrder, val2),
					i, rid)
				logLine(inventoryFile, reserveTS, "INF", "Message received",
					fmt.Sprintf(`event_key=inventory.reserved offset=%d topic=outbox.events value=%q`, offsetInventory, val2),
					i, rid)
			} else {
				logLine(orderFile, reserveTS, "INF", "Message received",
					fmt.Sprintf(`,"event_key":"inventory.reserved","offset":%d,"topic":"outbox.events","value":%q`, offsetOrder, val2),
					i, rid)
				logLine(inventoryFile, reserveTS, "INF", "Message received",
					fmt.Sprintf(`,"event_key":"inventory.reserved","offset":%d,"topic":"outbox.events","value":%q`, offsetInventory, val2),
					i, rid)
			}

			offsetOrder++
			offsetInventory++

			if r.Float64() < 0.55 {
				logLine(inventoryFile, reserveTS, "INF", "Ignoring event",
					"event=inventory.reserved", 0, rid)
			}
		}

		// payment.authorized
		if r.Float64() < 0.08 {
			val3 := fmt.Sprintf(`{"order_id":%d}`, i)

			if *format == "text" {
				logLine(orderFile, ts, "INF", "Message received",
					fmt.Sprintf(`event_key=payment.authorized offset=%d topic=outbox.events value=%q`, offsetOrder, val3),
					i, rid)
			} else {
				logLine(orderFile, ts, "INF", "Message received",
					fmt.Sprintf(`,"event_key":"payment.authorized","offset":%d,"topic":"outbox.events","value":%q`, offsetOrder, val3),
					i, rid)
			}

			offsetOrder++
		}
	}
}

func logLine(f *os.File, ts time.Time, level, msg string, extra string, orderID int, reqID string) {
	if *format == "text" {
		// TEXT FORMAT (your original style)
		line := fmt.Sprintf("%s %s %s", ts.Format(time.RFC3339), level, msg)
		if extra != "" {
			line += " " + extra
		}
		if reqID != "" {
			line += " request_id=" + reqID
		}
		if orderID != 0 {
			line += fmt.Sprintf(" order_id=%d", orderID)
		}
		line += "\n"
		_, _ = f.WriteString(line)
		return
	}

	// JSON FORMAT (default)
	req := ""
	if reqID != "" {
		req = fmt.Sprintf(`,"request_id":"%s"`, reqID)
	}

	line := fmt.Sprintf(
		`{"ts":"%s","level":"%s","msg":"%s"%s%s,"order_id":%d}`+"\n",
		ts.Format(time.RFC3339),
		level,
		msg,
		extra,
		req,
		orderID,
	)
	_, _ = f.WriteString(line)
}

func reqID(r *rand.Rand, orderID int) string {
	return fmt.Sprintf("req-%x-%x", orderID, r.Uint64())
}

func randomItems(r *rand.Rand) string {
	n := 1 + r.Intn(4)
	s := "["
	for i := 0; i < n; i++ {
		if i > 0 {
			s += ","
		}
		price := ""
		if r.Float64() < 0.75 {
			price = fmt.Sprintf(`,"price":%d`, 10+r.Intn(500))
		}
		s += fmt.Sprintf(`{"product_id":%d,"quantity":%d%s}`,
			1+r.Intn(20),
			1+r.Intn(5),
			price,
		)
	}
	return s + "]"
}
