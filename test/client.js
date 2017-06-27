var net = require('net');

var HOST = 'localhost';
var PORT = 9781;

var client = new net.Socket();
client.connect(PORT, HOST, function(test) {

    console.log('CONNECTED TO: ' + HOST + ':' + PORT);
    // Write a message to the socket as soon as the client is connected, the server will receive it as message from the client
    //client.write('I am Chuck Norris!');
    var dataPacket = "GTVT866104024682547#MC,,A,2314.256987448653,S,09076.80130004882,E,41.10,000,,,,N*7E|010000|#\n";
    //var dataPacket = "GTVT866104024682547#MC,,A,2316.6244944203537,S,09073.539733886719,E,41.10,000,,,,N*7E|010000|#\n";
    client.write(dataPacket);
    // for(j=0; j<=1000; j++) {
    //   console.log(j);
    // }
    // client.write(Buffer.from('78780a134500640001067404d60d0a', 'hex'));
    // for(i=0; i<=1000; i++) {
    //   console.log(i);
    // }
    // client.write(Buffer.from('78781f12110317122c0cc5028c633009b3b4a50034f101d601522800104e067235420d0a', 'hex'));
});
//78781f12110317160938c4028c528409b367cc00349f01d6015228003f30007b00ee0d0a
// Add a 'data' event handler for the client socket
// data is what the server sent to this socket
client.on('data', function(data) {
    console.log('DATA: ' + data);
    // Close the client socket completely
    //client.destroy();

});

// Add a 'close' event handler for the client socket
client.on('close', function() {
    console.log('Connection closed');
});

// for GPS data
// 78781f1211021200390fc5028ddca009b1dfa007547d01d601521a00d47e00028e3b0d0a

//23.074046463246766,90.70243835449219|23.216106724564785,90.70930480957031|23.171926338284234,90.80474853515625|23.276042451248856,90.82191467285156|23.21989293541487,90.92628479003906

//inner circle
//23.14256987448653,90.76801300048828|23.148883635455956,90.7796859741211|23.139412882477405,90.78998565673828|23.134045825457704,90.77522277832031
