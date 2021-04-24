package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/huobirdcenter/huobi_golang/pkg/client"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
)

const (
	SYMBOL = "dogeusdt"

	MULTI = "MULTI"
	EXEC  = "EXEC"
	HMSET = "HMSET"
	KEY   = "history"

	REDISHOST = "127.0.0.1:6379"
	HUOBIHOST = "api.huobi.pro"
)

var (
	duration = 30 * time.Second
)

func main() {
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)

	log.SetFlags(log.Ltime | log.Lshortfile)

	log.Println("server start...")
	defer log.Println("quit")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := redis.Dial("tcp", REDISHOST)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		ticker := time.NewTicker(duration)

		marketClient := new(client.MarketClient).Init(HUOBIHOST)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				resp, err := marketClient.GetHistoricalTrade(SYMBOL, market.GetHistoricalTradeOptionalRequest{Size: 2000})
				if err != nil {
					log.Println(err)
					continue
				}

				if err := c.Send(MULTI); err != nil {
					log.Println(err)
					continue
				}

				for _, data := range resp {

					for _, d := range data.Data {
						val, err := json.Marshal(d)

						if err != nil {
							log.Println(err)
							continue
						}

						t := time.Unix(d.Ts / 1000, 0)
						f := t.Format("2006-01-02")

						if err := c.Send(HMSET, fmt.Sprintf("%s-%s", KEY, f), d.TradeId, val); err != nil {
							log.Println(err)
						}
					}
				}

				if _, err := c.Do(EXEC); err != nil {
					log.Println(err)
				}
			}
		}
	}()

	<-quit
}