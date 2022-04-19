// Copyright © 2022 zc2638 <zc2638@qq.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"math/rand"
	"time"
)

const (
	// 守护线程数
	_daemonThreadNum = 2
	// 最小的请求间隔，_sleepMinMillSec+rand.Int63n(_sleepMinMillSec/2)
	_sleepMinMillSec = 200
)

var StopDaemonThread bool

var WrapFun = func(do func() error) {
	for i := 0; i < _daemonThreadNum; i++ {
		go func() {
			WaitStart()
			for {
				if StopDaemonThread {
					return
				}
				_ = do()
				time.Sleep(time.Duration(_sleepMinMillSec+rand.Int63n(_sleepMinMillSec/2)) * time.Millisecond)
			}
		}()
	}
}

func LoopRun(num int, f func()) {
	for i := 0; i < num; i++ {
		f()
	}
}

func WaitStart() {
	for n := time.Now(); n.Before(time.Date(n.Year(), n.Month(), n.Day(), 5, 59, 59, 899999999, n.Location())); n = time.Now() {
		time.Sleep(1 * time.Microsecond)
	}
}
