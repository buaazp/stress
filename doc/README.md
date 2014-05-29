## HTTP压测工具stress发布

[@招牌疯子](http://weibo.com/819880808) zp@buaa.us  

#### 概述

stress是一个HTTP测试工具，采用Go语言编写，它是从[vegeta](https://github.com/tsenart/vegeta)项目改造而成的，但是由于和vegeta的理念不同，底层改动较大，而且以后会向不同的方向发展，所以我将其重新命名，作为新的开源软件进行维护。  

Github地址：[https://github.com/buaazp/stress](https://github.com/buaazp/stress)

#### 目标

stress的设计目标是支持HTTP协议的各种测试需求，尤其以压力测试、稳定性测试为主，同时可以输出美观的测试结果，便于后期统计和对比。  
如果你用过ab，常常会苦恼于它只能对一个固定的请求进行压力测试，如果能有一个工具可以对一系列的请求进行测试，则更能反应出应用程序对外服务时所要面临的情况，那么stress就是你所要寻找的东西。

#### 功能

stress拥有vegeta的全部功能，同时增加了一些用起来更加顺手的东西，下面将一一介绍。

- 按指定速率进行压力测试
	
	`-rate`参数可以指定每秒发送多少个请求，`-duration`参数指定持续时间，用来构造一次恒定速率的测试。`-rate`值并非可以随意设置，它的上限取决与机器性能和操作系统的文件打开数限制。
	
- 按指定并发数进行压力测试
	
	此模式与ab工具相同，`-c`表示并发数，`-n`指定请求数，用来构造一次模拟高并发情形的压力测试。并发模式与速率模式冲突，不可同时进行。并发模式好比一次启动一定数量的线程，每个线程都在依次地发送请求，直到所有请求发完；而速率模式是恒定每隔一定时间（= 1s / rate）发一个请求出去，这个请求消耗多少时间不会影响下一个请求的发送。

- 自定义header

	有两种方式进行自定义header：全局header和局部header，两者可同时生效。  
	全局header在stress attack命令后加参数`-header="host:www.baidu.com"`进行设定，一旦设定之后本次测试中的所有请求都会带有此header。示例：
	
	````
stress attack -header="client:iPhone5S" -targets=down.txt -c=40 -n=100
	````
	
	局部header只针对单条测试请求生效，设定方法是在METHOD之后直接写入KV对即可，示例：
	
	````
GET client:iPhone5S resize-type:square http://127.0.0.1:8088/6xxkqpcm7j20b40e7myz.jpg
	````
	
- GET请求支持MD5校验

	在很多测试场景中，我们不能只依靠服务器返回的HTTP status code来判断请求是否正确，比如要测试图片裁剪，虽然返回200 OK了，但是图片不对，依然需要标记为错误，因此stress增加了请求结束后的MD5校验功能，使用方法是在URL之后设定以md5:开头的预期MD5值，示例：
	
	````
GET http://127.0.0.1:4869/5f189d8ec57f5a5a0d3dcba47fa797e2 md5:5f189d8ec57f5a5a0d3dcba47fa797e3
	````
	
	如果在MD5校验中失败，该请求的结果会被标记为一个特殊的结果码`250`，stress会认为结果码为`250`的case为MD5校验失败。


- POST请求支持以 binary/mutipart-form 形式上传文件

	如果要测试POST请求发送数据，stress支持在URL之后设定文件名，例如：
	
	````
POST http://127.0.0.1:4869/ password.txt
	````
	
	如果要以form表单形式上传文件，则在文件名之前加`form`关键字即可：
	
	````
POST http://127.0.0.1:4869/ form:5f189.jpeg
	````
	
	如果需要设定form表单中的文件名关键字（默认为filename，具体内容在RFC1867协议中），可以这样构造：
		
	````
POST http://127.0.0.1:4869/ form:yourfilename:5f189.jpeg
	````
	
- 设定测试请求来源

	默认来自标准输入stdin，因此单条测试请求可以直接通过管道输入给stress，例如：  
	
	````
echo "GET http://127.0.0.1:8088/6xxkqpcm7j20b40e7myz.jpg" | stress attack  -c=30 -n=1000
	````
	
	也可以将一系列的请求写在文件中，stress通过`-targets`参数打开目标文件进行测试。请求文件`down2.txt`示例：
	
	````
GET HOST:ww4.sinaimg.cn resize-type:wap320 http://127.0.0.1:8088/large/a13a0128jw1e6xxkqpcm7j20b40e7myz.jpg md5:ee08f10750475ad209a822ffe24f4e78
POST http://127.0.0.1:4869/ form:filename:5f189.jpeg
GET http://127.0.0.1:4869/5f189d8ec57f5a5a0d3dcba47fa797e2 md5:5f189d8ec57f5a5a0d3dcba47fa797e3
...
	````
	
	请求文件没有大小限制，构造成百上千条请求进行测试毫无压力。然后在stress attack命令中指定该文件即可开始测试：
	
	````
stress attack -targets=down2.txt -c=40 -n=10000
	````
	
	target文件中的请求默认会以随机顺序进行测试，如有必要可以单独设定发送顺序为依次执行，加上`-ordering="sequential"`即可，示例：
	
	````
stress attack -targets=down2.txt -c=40 -n=10000 -ordering="sequential"
	````
	
- report工具

	stress attack工具是测试工具，stress report是结果输出工具。在每次attack测试结束之后，原始测试结果默认写入result.json文件中，当然你也可以通过增加`-output="result.json"`参数进行自定义。如果你保存了多次测试的数据，可以通过report工具来转换成更容易阅读的格式：
	
	````
stress report -input=result.json,result2.json,result3.json -output=output.json -reporter=json
	````
	
	`-reporter`支持三种格式的输出`[text, json, plot]`，满足不同的需求。
	
- 多核支持

	为了更好地利用多核的优势，stress支持设定使用核心数，可使用的CPU核心越多，并发测试速度越快，示例如下：
	
	````
stress -cpus=4 attack -targets=down2.txt -c=100 -n=10000
	````

#### 展望

虽然现在的stress已经可以方便地使用，但是我心中还有一些对后续版本的期望。

- 增强型数据校验

	MD5校验无法适用于动态生成的网页（如包含了时间信息和用户信息的返回结果），最好是支持结果内的关键字匹配。这个功能实现容易，但是会严重降低压力测试的效率，所以还没有加入。

- 动态生成请求

	有这么一种特殊的需求，请求地址是符合某种规则的一系列地址（例如不同分辨率的图片请求），除了自己生成然后全部写到targets文件中这种笨办法外，最好可以是一条包含正则表达式匹配的请求，由stress动态生成并进行测试。  
	再不济，也要支持随机请求，可以称为“乱入模式”，stress构造随机请求压往服务器，看服务器是否出现意外错误。

#### 总结

stress可以用来做并发压力测试（`-c -n`模式），也可以用来做长时间的稳定性、准确性测试（`-rate -duration`模式），测试请求构造简单，功能丰富，尤其是还支持批量测试和数据校验，测试结果美观易读，可以说是测试HTTP服务的一把利器。欢迎试用和反馈。
	
	
	