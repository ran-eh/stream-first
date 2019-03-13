package mocks

import "github.com/stretchr/testify/mock"

type MockPubsub struct {
	mock.Mock
}

func (ps *MockPubsub) Sub(topics ...string) chan interface{} {
	return make(chan interface{})
}

func (ps *MockPubsub) Pub(msg interface{}, topics ...string) {
	ps.Called(msg, topics)
}

type MockRand struct {
	MockResult float64
}

func (r MockRand) Rand() float64 {
	return r.MockResult
}
