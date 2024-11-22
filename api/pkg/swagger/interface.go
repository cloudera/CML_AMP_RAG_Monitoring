package swagger

type SerializableParamsGetter interface {
	GetSerializedParams() ([]byte, error)
}

type SerializablePayloadGetter interface {
	GetSerializedPayload() ([]byte, error)
}

type CodeGetter interface {
	Code() int
}

type SwaggerResponse interface {
	CodeGetter
	SerializablePayloadGetter
}

type SwaggerParams interface {
	SerializableParamsGetter
}
