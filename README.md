# fikonfarm

fikonfarm is a software-defined Fibre Channel storage network. It will support FCP (SCSI) and FICON (ECKD and tape).

Translated from Swedish it means "Fig farm", and is [pronounced](https://translate.google.com/translate_tts?ie=UTF-8&q=Fikonfarm&tl=sv&total=1&idx=0&textlen=9&client=tw-ob) something like feecon-farm. It is a play on words
where the mainframe protocol FICON sounds like fig in Swedish. The farm part plays on that there is loads of FICONs.

## How it works

By using FCIP and FCoE as transport all hardware that is needed is a decently fast Ethernet network card.

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
 
