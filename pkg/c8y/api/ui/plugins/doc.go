// Package plugins covers uI plugins (web application extensions): upload a
// plugin zip, list, activate and resolve plugins by name.
//
// Reach it from a client with client.UIPlugins.
//
// Example, with a client and ctx in scope:
//
//	for plugin, err := range client.UIPlugins.ListAll(ctx, plugins.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(plugin.Get("name").String())
//	}
package plugins
