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

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/zc2638/ddshop/cmd/ddshop/app"

	"github.com/sirupsen/logrus"
)

const TimeFormat = "2006/01/02 15:04:05"

func main() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		FullTimestamp:          true,
		TimestampFormat:        TimeFormat,
	})
	command := app.NewRootCommand()
	time.Sleep(NextDay(5, 59, 40))
	fmt.Println(time.Now().String())
	//time.Sleep(5*time.Second)
	//fmt.Println(time.Now().String())
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}

func NextDay(hour, min, second int) time.Duration {
	n := time.Now()
	d := time.Date(n.Year(), n.Month(), n.Day(), hour, min, second, 0, n.Location())
	for !d.After(n) {
		d = d.AddDate(0, 0, 1)
	}
	return time.Until(d)
}
