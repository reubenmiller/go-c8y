// Package pagination provides client side pagination. ListAll methods return
// an Iterator which fetches pages on demand, with the page size, a maximum
// number of items, the paging strategy (offset, or time or id keyset) and
// parallel page fetches set through PaginationOptions.
//
// Reach it through the ListAll method of a service, for example
// client.Alarms.ListAll.
//
// Example, with a client and ctx in scope:
//
//	opts := managedobjects.ListOptions{
//		PaginationOptions: pagination.PaginationOptions{PageSize: 100, MaxItems: 1000},
//	}
//	it := client.ManagedObjects.ListAll(ctx, opts)
//	for mo, err := range it.Items() {
//		if err != nil {
//			return err
//		}
//		fmt.Println(mo.Get("id").String())
//	}
package pagination
