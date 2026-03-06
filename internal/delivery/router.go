package delivery

import (
	"context"
	"log"

	"github.com/shazmughal/orderbook/internal/metrics"
	"github.com/shazmughal/orderbook/internal/model"
)

// Router dispatches triggered alerts to the appropriate channel handlers.
type Router struct {
	handlers map[string]ChannelHandler
}

// ChannelHandler sends an alert via a specific delivery channel.
type ChannelHandler interface {
	Type() string
	Send(ctx context.Context, alert *model.TriggeredAlert, target string) error
}

func NewRouter(handlers ...ChannelHandler) *Router {
	hmap := make(map[string]ChannelHandler, len(handlers))
	for _, h := range handlers {
		hmap[h.Type()] = h
	}
	return &Router{handlers: hmap}
}

// Deliver sends the alert to all configured channels.
func (r *Router) Deliver(ctx context.Context, alert *model.TriggeredAlert) {
	for _, ch := range alert.DeliveryChannels {
		handler, ok := r.handlers[ch.Type]
		if !ok {
			log.Printf("delivery: no handler for channel type %q", ch.Type)
			metrics.DeliveryFailureTotal.WithLabelValues(ch.Type).Inc()
			continue
		}

		if err := handler.Send(ctx, alert, ch.Target); err != nil {
			log.Printf("delivery: %s to %s failed: %v", ch.Type, ch.Target, err)
			metrics.DeliveryFailureTotal.WithLabelValues(ch.Type).Inc()
		} else {
			metrics.DeliverySuccessTotal.WithLabelValues(ch.Type).Inc()
		}
	}
}
