## SubPub
### What is it?
It is another one implementation of subscriber-publisher
protocol, but with three main restrictions:
1. No message is lost
2. Slow subscribers don't slow down whole system
3. No goroutines are going to be leaked

It has created a lot of problems, most of which had pretty
disgusting solutions like "add another mutex".

### How it works?
On each subscription two new goroutines enter workflow.
One catches messages from broadcaster and adds them
to inner queue, and other one clears queue and process all
the messages in it. Let's call them receiver and processor.

Processor and receiver need to access inner queue concurrently,
so there is one mutex for that: processor takes it on grabbing
queue and clearing it and receiver for each append.

After processing queue processor goes to sleep on inner Cond,
waking up only by processor or Unsubscribe method. Because
of nature of Cond call, processor needs to check if length
of inner queue is still 0, but taking mutex every time would be
atrocious, so we have length of queue stored as atomic.Int64 in
subscription.

More on closing: we need to stop both goroutines. Receiver stopping
is really simple as its only job is to listen on channel. So
we close it. To stop processor I decided to add active field
to subscription, which is checked in Cond call in processor.
Another atomic variable.

That's all for subscriptions, let's move on to broadcasters.
I decided to stop on model 1 broadcaster:1 topic. So each 
broadcaster put every new subscriber inside its map locked by mutex.
I decided that sync.Map wouldn't fit, because flow of 
subscribers can be really fast.

And on the outermost layer resides subpub system, containing
sync.Map of broadcasters for all topics. Topic creation is 
a rare occasion(which might not be the case, but why not), so 
sync.Map is the best fit there.

### Usage
Look inside [domain.go](/domain.go) to find all about interfaces.

### Testing
```bash
go test {path to subpub pkg} -v -race
```