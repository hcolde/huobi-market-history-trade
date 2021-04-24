package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)

const (
	SCAN    = "SCAN"
	HKEYS   = "HKEYS"
	HGET    = "HGET"

	GRAPH = "#"

	STR = "[%s] %.2f%% %d/%d remain:%v          "
)

var (
	c redis.Conn

	redisHost string
)

type Bar struct {
	ctx context.Context

	percent int    //百分比
	cur     int    //当前进度位置
	total   int    //总进度
	graph   string //显示符号

	startTime time.Time // 开始时间

	CH chan int
}

func NewBar(ctx context.Context, total int, graph string, startTime time.Time) *Bar {
	bar := &Bar{
		ctx:     ctx,
		total:   total,
		graph:   graph,
		CH:      make(chan int),
		startTime: startTime,
	}

	go bar.show()
	return bar
}

func (bar *Bar) show() {
	var r interface{}
	for {
		select {
		case <-bar.ctx.Done():
			return
		case bar.cur = <-bar.CH:
			graph := ""

			n := bar.total / 100
			m := bar.cur / n

			for i := 0; i < 100; i++ {
				if i < m {
					graph += bar.graph
				} else {
					graph += " "
				}
			}

			now := time.Now()
			duration := now.Sub(bar.startTime)

			v := float64(bar.cur) / duration.Seconds()

			remain := float64(bar.total - bar.cur) / v

			r, _ = time.ParseDuration(fmt.Sprintf("%.0fs", remain))
			if r == "0s" {
				r = " "
			}

			s := fmt.Sprintf(STR, graph, float32(bar.cur) / float32(bar.total) * 100, bar.cur, bar.total, r)
			fmt.Printf("\r%s", s)
		}
	}
}

type Data struct {
	Amount    string `json:"amount"`
	TradeId   int64  `json:"trade-id"`
	Ts        int64  `json:"ts"`
	Id        string `json:"id"`
	Price     string `json:"price"`
	Direction string `json:"direction"`
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	flag.StringVar(&redisHost, "host", "127.0.0.1:6379", "redis host")
	flag.Parse()

	conn, err := redis.Dial("tcp", redisHost)
	if err != nil {
		log.Fatal(err)
	}

	c = conn

	keys, err := scan()
	if err != nil {
		log.Fatal(err)
	}

	if keys == nil {
		log.Fatal("not key")
		return
	}

	i := 1
	for key := range keys {
		fmt.Printf("\n%d/%d export:%s\n", i, len(keys), key)
		i++

		data, err := hGetAll(key)
		if err != nil {
			log.Println(key, err)
			continue
		}

		if data == nil {
			continue
		}

		f, err := os.Create(fmt.Sprintf("%s.csv", key))
		if err != nil {
			log.Println(err)
			return
		}

		c := csv.NewWriter(f)
		if err := c.WriteAll(data); err != nil {
			log.Println(err)
		}

		c.Flush()

		if err := f.Close(); err != nil {
			log.Println(err)
		}
	}
}

// 获取所有的key
func scan() (map[string]struct{}, error) {
	keys := make(map[string]struct{})

	cursor := 0

	for {
		reply, err := redis.Values(c.Do(SCAN, cursor))
		if err != nil {
			return nil, err
		}

		if len(reply) < 2 {
			return nil, nil
		}

		_cursor, err := redis.Int(reply[0], nil)
		if err != nil {
			return nil, err
		}

		cursor = _cursor

		value, err := redis.Strings(reply[1], nil)
		if err != nil {
			return nil, err
		}

		for _, v := range value {
			keys[v] = struct{}{}
		}

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func hGetAll(key string) ([][]string, error) {
	reply, err := redis.Strings(c.Do(HKEYS, key))
	if err != nil {
		return nil, err
	}

	result := make([][]string, 1)

	result[0] = []string{"序号", "时间戳", "交易量", "价格", "方向"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	data := Data{}

	bar := NewBar(ctx, len(reply), GRAPH, time.Now())

	for i, field := range reply {
		res, err := redis.Bytes(c.Do(HGET, key, field))
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(res, &data); err != nil {
			return nil, err
		}

		result = append(result, []string{
			strconv.Itoa(i),
			strconv.FormatInt(data.Ts, 10),
			data.Amount,
			data.Price,
			data.Direction,
		})

		bar.CH <- i
	}

	return result, nil
}
