#Project fsea

##概述
fsea提供一个基于桶文件的http文件服务。这个服务是单机的，无Cache的。

##目录结构
fsea的默认系统目录结构如下：

```
./bin/fsea
./conf/fsea.json
./buckets/fid_12.bkt
```
也可以通过启动命令来指定conf文件：

```
fsea --conf=/etc/fsea.conf
```

##配置文件格式
fsea的配置文件为toml文件格式。

```
Id = "fsea的服务标识"
Port = 8080
AdminIP = ["192.168.200.17", "192.168.200.18"]
Regto = ["http://server1/reg"]

[[Bucket]]
  Id = "0"
  Path = "/data/fsea/buckets"

  [[Bucket.File]]
    Id = "1"
    Name = "1_12.bkt"

[[Bucket]]
  Id = "1"
  Path = "/data/fsea/buckets2"

  [[Bucket.File]]
    Id = "0"
    Name = "0_12.bkt"

  [[Bucket.File]]
    Id = "1"
    Name = "1_12.bkt"

[Large]
  Path = "/data/fsea/large"
  Max = 0
  Fold = "week"
```
以下用`config.`来引用配置文件中配置的信息。

## 管理类Web API
管理类API提供了一系列管理接口，用于对配置的设置和修改。

这些接口仅能通过`config.admin_ip`指定的IP地址进行访问。

```
小贴士：
127.0.0.1虽然可以不设置在config.admin_ip的地址列表里，但系统内部允许访问。
```
管理API包括以下子项

```
/mount 挂载文件
/umount 卸载文件
/setconfig 设置配置值
/addconf 增加配置为列表的值
/delconf 删除配置为列表的其中给一个值

```


### /mount 挂载文件
```
/mount/[Bucket ID]/[Bucket Size]/[Bucket Count]
/mount/[Bucket ID]/[File Name]
```
#### 描述
`mount`命令在指定设置的目录下创建一个文件。这个文件是新建的

#### 参数

`Bucket ID`：由`config.bucket[].id`指定的目录；

`Bucket Size`：用于指定桶的大小，单位是4096字节。例如：25，则实际桶大小为25*4096，也就是100K大小。限定值域为1~2048，也就是最大8兆。

`Bucket Count`：用于指定该文件的桶个数。

`File Name`：如果指定的是`File Name`,那么就挂载已经存在的文件，并分配ID。如果该文件已经挂载系统中，则先解除挂载，再重新挂载。

#### 返回值（HTTP STATUS）

##### 200 OK
增加成功

```
{
	id: 目录id:文件id,
	size: 文件大小,
	name: 文件名（全路径）
}
```

#### 400 Bad Request

```
{
	Err: 101,
	Message: "Invalid Bucket ID",
	Detail: "Valid Bucket ID is [0, 1, 2]"
}
```
参数错误，error可以是以下值：

```
101 "Invalid Fold ID": 指定的Fold ID未配置
102 "Bucket Size Too Large": 指定的桶值过大。范围限定1~2048
103 "File Too Big": 根据Bucket Size * Bucket Count计算出的文件大小超过16G。
	考虑到文件复制、移动等因素，桶文件大小控制在16G以下。
104 "File Not Found": File Name指定的文件不存在。

```

#### 500 Internal Server Error
创建文件失败。错误码201，detail中会将创建过程中的具体错误信息输出。

```
{
	Err: 201,
	Message: "Create File Failed",
	Detail: ""
}
```

### /umount 卸载文件
```
/mount/[File ID]
```
#### 描述
从系统中移除一个文件。

`File ID`的格式为，`config.bucket.id:config.bucket.file.id`

#### 返回值

##### 400 Bad Request

```
{
	Err: 101,
	Message: "Invalid Fold ID",
	Detail: "Valid Fold ID is [0, 1, 2]"
}
```
参数错误，error可以是以下值：

```
101 "Invalid Bucket ID": 指定的Fold ID未配置
105 "Invalid File ID": File ID指定的文件未被挂载

```




