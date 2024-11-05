package channels

import "sync"

var FolderChannel chan map[string]string
var FileChannel chan map[string]string

var Mu sync.Mutex
var Wg sync.WaitGroup

var FolderChannelLock bool = false
var FileChannelLock bool = false