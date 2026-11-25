// Package pipeline provides composable helpers for streams of items, which are
// iter.Seq2[T, error] values like the iterators returned by ListAll: filter,
// batch, merge, throttle, and run a function for every item with bounded
// concurrency and error reporting.
//
// Use it with the iterators of the services.
//
// Example, with a client and ctx in scope:
//
//	items := client.ManagedObjects.ListAll(ctx, managedobjects.ListOptions{}).Items()
//	named := pipeline.Filter(items, func(mo jsonmodels.ManagedObject) bool {
//		return mo.Get("name").String() != ""
//	})
//	for batch, err := range pipeline.Batch(named, 10) {
//		if err != nil {
//			break
//		}
//		fmt.Println("batch of", len(batch))
//	}
package pipeline
