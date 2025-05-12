package subpub

import "errors"

var ErrClosed = errors.New("subpub: broker closed")
var ErrTopicNameIsEmpty = errors.New("subpub: topic name must not be empty")
var ErrTopictNotFound = errors.New("subpub: subject does not exist")
