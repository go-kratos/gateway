package source

import (
	"github.com/go-kratos/gateway/api"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/log"
)

// Option is an Rule option.
type Option func(*Rule)

// Logger with server logger.
func Logger(logger log.Logger) Option {
	return func(o *Rule) {
		o.log = log.NewHelper(logger)
	}
}

type Rule struct {
	cfg config.Config
	log *log.Helper
}

func NewRule(cfg config.Config, opts ...Option) *Rule {
	rule := &Rule{
		log: log.NewHelper(log.DefaultLogger),
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(rule)
	}

	return rule
}

func (r *Rule) Load() (*api.Rule, error) {
	var rule api.Rule
	err := r.cfg.Value("rule").Scan(&rule)
	if err != nil {
		return nil, err
	}
	r.log.Infof("%+v", rule)
	return &rule, nil
}

func (r *Rule) Watch(f func(rule *api.Rule)) error {
	return r.cfg.Watch("rule", func(k string, v config.Value) {
		var rule api.Rule
		err := v.Scan(&rule)
		if err == nil {
			f(&rule)
		} else {
			r.log.Info("watch key routeRule failed!")
		}
	})
}
