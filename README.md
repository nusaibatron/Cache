# Cache

NUSAIBA RAHMAN
UC BERKELEY CS61C 
All rights and property belong to UC Berkeley. 

A file server uses some protocol to connect to other computers on a network. 
When it gets a request from some other computer on the network, it processes 
the request (typically as a path to a file in the form of a URL), and returns 
back the data of the file which is local on the server. The protocol allows 
for a well defined communication method to exist between the different 
computers on the network. The common protocol which used to be used everywhere 
for this is http. 

The problem for many of these sites is that disk reads are really slow so 
when many users are trying to access a file, it can take a long time for the 
request to get a response. To combat this, many file servers implement caching 
so that subsequent reads to the same file can be faster.

One problem a server may face is a lot of disk reads at once. This can make
the response to a request take a long time. Sometimes the disk may just never 
respond though is highly unlikely. Because of this, servers implement a 
timeout system to ensure that requests are answered.

Go is the language of choice becuase it was designed to support concurrency.

This is the implementation of a file server's file cacher. 
