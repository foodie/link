package codec

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/funny/link"
)

//协议基本信息
type JsonProtocol struct {
	types map[string]reflect.Type //名字-反射类型
	names map[reflect.Type]string //反射类型-名字
}

//新建一个协议
func Json() *JsonProtocol {
	return &JsonProtocol{
		types: make(map[string]reflect.Type),
		names: make(map[reflect.Type]string),
	}
}

//注册一个基本的类型
func (j *JsonProtocol) Register(t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := rt.PkgPath() + "/" + rt.Name()
	j.types[name] = rt
	j.names[rt] = name
}

//按照名字注册基本类型
func (j *JsonProtocol) RegisterName(name string, t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	j.types[name] = rt
	j.names[rt] = name
}

//通过JsonProtocol创建一个新的link.Codec
func (j *JsonProtocol) NewCodec(rw io.ReadWriter) (link.Codec, error) {
	codec := &jsonCodec{
		p:       j,
		encoder: json.NewEncoder(rw),
		decoder: json.NewDecoder(rw),
	}
	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

//in信息
type jsonIn struct {
	Head string           // 头部信息
	Body *json.RawMessage //原生的json串
}

//out信息
type jsonOut struct {
	Head string //头部信息
	Body interface{}
}

type jsonCodec struct {
	p       *JsonProtocol //协议
	closer  io.Closer     //关闭
	encoder *json.Encoder //编码
	decoder *json.Decoder //解码
}

//接收信息，然后解压到body里面
func (c *jsonCodec) Receive() (interface{}, error) {
	var in jsonIn
	//解压消息
	err := c.decoder.Decode(&in)
	if err != nil {
		return nil, err
	}
	var body interface{}
	if in.Head != "" {
		if t, exists := c.p.types[in.Head]; exists {
			body = reflect.New(t).Interface()
		}
	}
	//解压信息
	err = json.Unmarshal(*in.Body, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

//发送信息，头部是消息的类型
func (c *jsonCodec) Send(msg interface{}) error {
	var out jsonOut
	t := reflect.TypeOf(msg)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if name, exists := c.p.names[t]; exists {
		out.Head = name
	}
	out.Body = msg
	return c.encoder.Encode(&out)
}

//关闭jsonCodec
func (c *jsonCodec) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
