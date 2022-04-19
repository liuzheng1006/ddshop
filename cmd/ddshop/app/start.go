package app

import (
	"errors"
	"fmt"
	"github.com/zc2638/ddshop/pkg/notice"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/zc2638/ddshop/core"
)

type Option struct {
	Cookie   string
	BarkKey  string
	Interval int64
}

const (
	// 程序并行数量
	_operateParallelNum = 1
	// 程序持续运行时间
	_programRunTime = 8 * time.Minute
)

func NewRootCommand() *cobra.Command {
	opt := &Option{}
	cmd := &cobra.Command{
		Use:          "ddshop",
		Short:        "Ding Dong grocery shopping automatic order program",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			session, err := prepare(opt)
			if err != nil {
				return err
			}

			start(session, opt)

			return monitor(opt)
		},
	}
	cmd.Flags().StringVar(&opt.Cookie, "cookie", "", "设置用户个人cookie")
	cmd.Flags().StringVar(&opt.BarkKey, "bark-key", "", "设置bark的通知key")
	cmd.Flags().Int64Var(&opt.Interval, "interval", 200, "设置请求间隔时间(ms)")
	return cmd
}

func prepare(opt *Option) (session *core.Session, err error) {
	if opt.Cookie == "" {
		err = errors.New("请输入用户Cookie")
		return
	}
	session = core.NewSession(opt.Cookie, opt.Interval)
	if err = session.GetUser(); err != nil {
		err = fmt.Errorf("获取用户信息失败: %v", err)
		return
	}
	if err = session.Choose(); err != nil {
		return
	}
	return
}

func start(session *core.Session, opt *Option) {
	for i := 0; i < _operateParallelNum; i++ {
		go func() {
			for {
				if core.StopDaemonThread {
					return
				}
				if err := flow(session); err != nil {
					switch err {
					case core.ErrorNoValidProduct, core.ErrNoValidFreight, core.ErrorNoReserveTime:
						logrus.Errorf("%+v，%d 秒后退出！", err.Error(), 10)
						time.Sleep(10 * time.Second)
						errCh <- err
						return
					default:
						logrus.Error(err)
						time.Sleep(time.Duration(opt.Interval+rand.Int63n(opt.Interval/2)) * time.Millisecond)
					}
				}
			}
		}()
		time.Sleep(400 * time.Millisecond)
	}
}

func monitor(opt *Option) error {
	ticker := time.NewTicker(_programRunTime)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		return fmt.Errorf("程序执行%d分钟退出", _programRunTime)
	case err := <-errCh:
		return err
	case <-successCh:
		core.LoopRun(10, func() {
			logrus.Info("抢菜成功，请尽快支付!")
		})
		if opt.BarkKey == "" {
			return fmt.Errorf("Bark消息Key为nil")
		}
		ins := notice.NewBark(opt.BarkKey)
		for i := 0; i < 120; i++ {
			if err := ins.Send("抢菜成功", "叮咚抢菜成功，请尽快支付！"); err != nil {
				logrus.Warningf("Bark消息通知失败: %v", err)
			}
			time.Sleep(2 * time.Second)
		}
		return nil
	}
}
