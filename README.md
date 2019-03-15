# Stream First Coding Exercise
## General
### Stream First Architecture
For a while now, I have been interested in 
[Stream First Architecture](
https://mapr.com/ebooks/intro-to-apache-flink/chapter-2-streaming-first-architecture.html)
for pipeline application, and this coding exercise seemed like a good opportunity to try it out.  
Some properties of Stream First Architecture are
* Components communicate via a robust pub/sub message transport, rather 
than via (remote or local) procedure calls.
* There are no central data stores acting as central sources of truth.  
Instead, there are shared, ever moving event streams and their history.  
Each component maintains its own local data store.

It is claimed that this architecture gives rise to systems that are 
easier to create, scale and maintain highly complex, highly performant systems.  **If you prefer 
a more traditional call/return structure, let me know and I will refactor the code accordingly.** 
### System Architecture
The application is composed of the following components
* The **Simulated New Order Service** published a stream of new order 
events, to be consumed by the Kitchen component downstream.
* The **Shelf Service** maintains the contents of the shelves.  It consumes New order events, 
order pickup events and order expiration events.  and  trigger it to store prepared order on shelves.
It also consumed.  It produces shelving, re-shelving (overflow to primary) and waste events. 
* The **Simulated Pickup Service** consumes shelving events, and for every newly shelved order 
it produced a pickup event after a random time.  It also observes order expiration events and cancels 
pending pickups as needed.
* The **Shelf Life Service** observes shelving/re-shelving events and produces
order value events anb order expiration events as needed.  It also observes pickup events and stops 
monitoring for picked up orders.
* The **UI Service** consumes value events, and removed orders from display when orders are 
picked up or become expired.
### Tech Stack
* Go was chosen as the programming language.  Light weight threads (goroutines) and comm channels
 are native, making async comms natural and easy to implement.
* The transport layer is implemented using the [cskr/pubsub](https://github.com/cskr/pubsub) library.
* For unit tests, the [testify](https://github.com/stretchr/testify) framework is used, as I find it 
more natural and quicker to code than the native unit tests.
* To keep UI simple, I use the [goterm](https://github.com/buger/goterm) console rendering library. 
