package core

type CheckoutRequest struct {
	policy     string
	fromclient string
	//fromclient *Client
	available int64
	count     int
	response  chan CoAck
}
