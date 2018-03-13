package codec

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/funny/link"
)

type JsonProtocol struct {
	types map[string]reflect.Type //string-定义类型
	names map[reflect.Type]string //类型-string
}

//初始化json协议
func Json() *JsonProtocol {
	return &JsonProtocol{
		types: make(map[string]reflect.Type),
		names: make(map[reflect.Type]string),
	}
}

//注册任意类型
func (j *JsonProtocol) Register(t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	name := rt.PkgPath() + "/" + rt.Name()
	j.types[name] = rt //名字(string):获取包+类型reflect
	j.names[rt] = name //获取包+类型reflect: 名字(string)
}

//注册一种类型，根据名字
func (j *JsonProtocol) RegisterName(name string, t interface{}) {
	rt := reflect.TypeOf(t)
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	j.types[name] = rt
	j.names[rt] = name
}

func (j *JsonProtocol) NewCodec(rw io.ReadWriter) (link.Codec, error) {
	codec := &jsonCodec{
		p:       j,
		encoder: json.NewEncoder(rw),
		decoder: json.NewDecoder(rw),
	}
	codec.closer, _ = rw.(io.Closer)
	return codec, nil
}

type jsonIn struct {
	Head string
	Body *json.RawMessage
}

type jsonOut struct {
	Head string
	Body interface{}
}

type jsonCodec struct {
	p       *JsonProtocol
	closer  io.Closer
	encoder *json.Encoder
	decoder *json.Decoder
}

func (c *jsonCodec) Receive() (interface{}, error) {
	var in jsonIn
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
	err = json.Unmarshal(*in.Body, &body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

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

//关闭json连接
func (c *jsonCodec) Close() error {
	if c.closer != nil {
		return c.closer.Close()
	}
	return nil
}
