# huobi-market-history-trade

> 获取火币网指定交易的近期所有交易记录

主程序会将交易信息缓存到redis中，再通过export程序将信息导出到csv文件中。



⚠️导出程序预计需要占用300M左右到内存



## 说明

* 缓存信息格式为

  ```json
  {
      "amount":"2878.94",
      "trade-id":100096058215,
      "ts":1619129827295,
      "id":"102259024623261492985671044",
      "price":"0.273405",
      "direction":"buy"
  }
  ```

  * `amount` - 以基础币种为单位的交易量
  * `trade-id` - 唯一成交id
  * `ts` - 调整为新加坡时间的时间戳，单位毫秒
  * `price` - 以报价币种为单位的成交价格
  * `direction` - 交易方向：“buy” 或 “sell”, “buy” 即买，“sell” 即卖

* 导出csv格式为

  | 序号 | 时间戳        | 交易量  | 价格     | 方向 |
  | ---- | ------------- | ------- | -------- | ---- |
  | 1    | 1619129827295 | 2878.94 | 0.273405 | buy  |



## 使用

* 使用golang编译

  1. 拷贝该项目至外网服务器（火币网api在大陆无法访问）

     ```shell
     git clone https://github.com/hcolde/huobi-market-history-trade.git
     ```

     

  2. 编译

     ```shell
     > go build export/main.go
     > mv main export_main
     > go build main.go
     > chmod +x main && chmod +x export_main
     ```

     

  3. 运行

     1. 获取数据

        ```shell
        nohup ./main -host "127.0.0.1:6379" -symbol "btcusdt" > log 2>&1 &
        ```

        * `host` - redis服务器地址
        * `symbol` - 交易代码

     2. 导出数据

        ```shell
        ./export_main
        ```

* 下载可执行文件

  https://github.com/hcolde/huobi-market-history-trade/releases/tag/1.0

