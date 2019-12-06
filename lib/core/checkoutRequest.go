package core

// CheckoutRequest Object used by worker to request a workunit
type CheckoutRequest struct {
	policy     string
	fromclient string
	//fromclient *Client
	available int64
	count     int
	response  chan CoAck
}
