# fikonfarm

fikonfarm is a software-defined Fibre Channel storage network. It will support FCP (SCSI) and FICON (ECKD and tape).

Translated from Swedish it means "Fig farm", and is [pronounced](https://translate.google.com/translate_tts?ie=UTF-8&q=Fikonfarm&tl=sv&total=1&idx=0&textlen=9&client=tw-ob) something like feecon-farm. It is a play on words
where the mainframe protocol FICON sounds like fig in Swedish. The farm part plays on that there is loads of FICONs.

## How it works

There are three possible integration ways planned, all with pros/cons:

 * FCIP: Fibre Channel over IP is straight forward conceptually, but is implemented in proprietary ways and thus require reverse engineering
 * FCoE: Fibre Channel over Ethernet might work, but there is little to no precedence of running FICON over it.
 * FICON: Using an FPGA card like the [DE5-Net](https://www.ebay.com/sch/i.html?_nkw=de5-net) it would be possible to send/receive FICON natively, but costs are higher to get the card
 
Right now the FCIP support for Brocade 7800 is looking promising, and it is planned that the DE5-Net card will be supported. FCoE will be supported as well, but possibly only for FCP - we will see what happens down the line.

An example integration would look something like this:

![fikonfarm integration](https://docs.google.com/drawings/d/e/2PACX-1vS2aFTFT3xdmoWAx8AF30wG4gXG4b6XXCxEAwZUj7z-cbVBQGwZxBw47zqYpyHu7R7SOFv8rNZfeUaO/pub?w=723&amp;h=382)

## Motivation

As a mainframe hobbyist FICON storage is a real bummer. I want to spend time tinkering with the mainframe hardware
and OSes like z/OS and z/VM, not spend money and time finding [old disk arrays in unknown state](https://blog.mainframe.dev/2019/06/unboxing-accessories-and-ds6800-troubles.html) on eBay.

Given that [Hercules](http://Hercules-390.org) has managed to emulate ECKD disks pretty well, hopefully this project
is doable.

## Non-goals

 * Do not expect enterprise grade performance
 * It will not be redundant, at least initially
 
