var net = require('net');

var HOST = '127.0.0.1';
var PORT = 6969;

var client = new net.Socket();
client.connect(PORT, HOST, function(test) {

    console.log('CONNECTED TO: ' + HOST + ':' + PORT);
    // Write a message to the socket as soon as the client is connected, the server will receive it as message from the client
    //client.write('I am Chuck Norris!');
    var hexDataPacket = Buffer.from('78780d0103588990581855990001c9170d0a', 'hex');
    //var hexDataPacket = Buffer.from('78780d0102588990581855990001e5300d0a', 'hex');
    //var hexDataPacket = Buffer.from('78780d010158899058185599000191590d0a', 'hex');
    //78781f1211030b0f2302c9028dc4dc09b388c00054c201d601521500b401005670150d0a
    //78781f1211030b0f2305c9028dc4dc09b388c000d4c201d6015217006792005821190d0a
    //78781f1211030b0f2309ca028dc4e409b3887000546201d6015217006792005a6fdf0d0a
    client.write(hexDataPacket);
    for(j=0; j<=1000; j++) {
      console.log(j);
    }
    client.write(Buffer.from('78780a13060604000200027ff20d0a', 'hex'));
    for(i=0; i<=10000; i++) {
      console.log(j);
    }
    //client.write(Buffer.from('78781f1211030b0f2302c9028dc4dc09b388c00054c201d601521500b401005670150d0a78781f1211030b0f2305c9028dc4dc09b388c000d4c201d6015217006792005821190d0a78781f1211030b0f2309ca028dc4e409b3887000546201d6015217006792005a6fdf0d0a', 'hex'));
    //client.write(Buffer.from('78781f1211030b0f2302c9028dc4dc09b388c00054c201d601521500b401005670150d0a', 'hex'));
    setInterval(function() {
        client.write(Buffer.from('78780a13060604000200027ff20d0a', 'hex'));
    }, 5000);
    setInterval(function() {
        client.write(Buffer.from('78781f1211021a143228c7028d47a009b207a000c52001d6015224003ccb0000b6a00d0a', 'hex'));
    }, 3000);
    setInterval(function() {
        client.write(Buffer.from('7878251611021a143228c7028d47a009b207a00045200901d6015224003ccb1006040202000512dc0d0a', 'hex'));
    }, 10000);
});
//7878251611021a143228c7028d47a009b207a00045200901d6015224003ccb1006040202000512dc0d0a  --- actual device
//787825160B0B0F0E241DCF027AC8870C4657E60014020901CC00287D001F726506040101003656A40D0A  --- sample

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
