// Package event provides the Solar plugin event system.
//
// Events are multi-subscriber, priority-ordered, and support cancellation
// via Context.Cancel() and mutation via pointer fields. The server fires
// events at specific hook points; plugins subscribe via the global event
// instances.
//
//	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, data plugin.PlayerChatData) {
//	    if strings.Contains(*data.Message, "spam") {
//	        ctx.Cancel()
//	    }
//	}, plugin.PriorityNormal)
package event
