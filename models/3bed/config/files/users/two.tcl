set ns [new Simulator]
source tb_compat.tcl

set a [$ns node]
set b [$ns node]
set lnk [$ns duplex-link $a $b 10000Mb 0ms DropTail]

$ns rtproto Static
$ns run

