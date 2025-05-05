## SubPub
### What is it?
It is another one implementation of subscriber-publisher
protocol, but with four main restrictions:
1. No message is lost
2. Slow subscribers don't slow down whole system
3. No goroutines are going to be leaked
4. System should be able to stop gracefully, but considering 
passed context. 

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

Closing is kinda interesting too. After context close no more 
than 1 handler will be killed. Others will just hang there.
Caller will have no way to kill them, as system will be marked
as closed to prevent misusing.

### Why no interfaces on inside?
As you might see, inner structures separated enough to be
easily abstracted by using interfaces, but the last step
has not been done. There are two main reasons:
1. Testing did not require it.
2. These interfaces would have only 1 implementation.

I see no other needs to use interfaces in that project.
But once again, this could easily be done in no more than
5 minutes.

### Usage
Look inside [domain.go](./domain.go) to find all about interfaces.

### Testing
```bash
go test {project root}/pkg/subpub -v -race
```