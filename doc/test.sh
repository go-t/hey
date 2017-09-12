#!/bin/bash

URL='http://'

for opt in "-d" "--data-binary" "--data-raw" "--data-urlencode" ; do
 echo curl $opt "a.txt"    $opt "a+txt"    "$URL" && curl --trace-ascii - $opt "a.txt"     $opt "a+txt"    "$URL"
 echo curl $opt "=a.txt"   $opt "=a+txt"   "$URL" && curl --trace-ascii - $opt "=a.txt"    $opt "=a+txt"   "$URL"
 echo curl $opt "b=a.txt"  $opt "b=a+txt"  "$URL" && curl --trace-ascii - $opt "b=a.txt"   $opt "b=a+txt"  "$URL"
 echo curl $opt "@a.txt"   $opt "@a+txt"   "$URL" && curl --trace-ascii - $opt "@a.txt"    $opt "@a+txt"   "$URL"
 echo curl $opt "b@a.txt"  $opt "b@a+txt"  "$URL" && curl --trace-ascii - $opt "b@a.txt"   $opt "b@a+txt"  "$URL"
 echo curl $opt "b=@a.txt" $opt "b=@a+txt" "$URL" && curl --trace-ascii - $opt "b=@a.txt"  $opt "b=@a+txt" "$URL"
done
