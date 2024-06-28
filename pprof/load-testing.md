```Bash
eochieng@eochieng-thinkpadp16vgen1:~$ hey -c 1 -z 30s -m GET https://localhost:8083/catalogs/operatorhubio/all.json

Summary:
  Total:	30.0579 secs
  Slowest:	0.2741 secs
  Fastest:	0.1303 secs
  Average:	0.1466 secs
  Requests/sec:	6.8202
  
  Total data:	26788102965 bytes
  Size/request:	130673673 bytes

Response time histogram:
  0.130 [1]	|
  0.145 [147]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.159 [46]	|■■■■■■■■■■■■■
  0.173 [1]	|
  0.188 [0]	|
  0.202 [0]	|
  0.217 [0]	|
  0.231 [0]	|
  0.245 [0]	|
  0.260 [0]	|
  0.274 [10]	|■■■


Latency distribution:
  10% in 0.1350 secs
  25% in 0.1366 secs
  50% in 0.1392 secs
  75% in 0.1458 secs
  90% in 0.1506 secs
  95% in 0.2598 secs
  99% in 0.2735 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.1303 secs, 0.2741 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0004 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0000 secs
  resp wait:	0.0005 secs, 0.0004 secs, 0.0009 secs
  resp read:	0.1460 secs, 0.1298 secs, 0.2727 secs

Status code distribution:
  [200]	205 responses



eochieng@eochieng-thinkpadp16vgen1:~$
```



```Bash
eochieng@eochieng-thinkpadp16vgen1:~$ hey -c 5 -z 30s -m GET https://localhost:8083/catalogs/operatorhubio/all.json

Summary:
  Total:	30.4993 secs
  Slowest:	1.3598 secs
  Fastest:	0.5231 secs
  Average:	0.7070 secs
  Requests/sec:	7.0494
  
  Total data:	28094839695 bytes
  Size/request:	130673673 bytes

Response time histogram:
  0.523 [1]	|
  0.607 [2]	|■
  0.690 [148]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.774 [54]	|■■■■■■■■■■■■■■■
  0.858 [0]	|
  0.941 [0]	|
  1.025 [0]	|
  1.109 [0]	|
  1.193 [0]	|
  1.276 [2]	|■
  1.360 [8]	|■■


Latency distribution:
  10% in 0.6516 secs
  25% in 0.6665 secs
  50% in 0.6792 secs
  75% in 0.6959 secs
  90% in 0.7106 secs
  95% in 1.2656 secs
  99% in 1.3545 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0002 secs, 0.5231 secs, 1.3598 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0005 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.0191 secs, 0.0007 secs, 0.0373 secs
  resp read:	0.6877 secs, 0.5064 secs, 1.3291 secs

Status code distribution:
  [200]	215 responses



eochieng@eochieng-thinkpadp16vgen1:~$ 
`





```Bash
eochieng@eochieng-thinkpadp16vgen1:~$ hey -c 20 -z 30s -m GET https://localhost:8083/catalogs/operatorhubio/all.json

Summary:
  Total:	32.1419 secs
  Slowest:	4.6349 secs
  Fastest:	2.1413 secs
  Average:	3.2346 secs
  Requests/sec:	6.0668
  
  Total data:	25481366235 bytes
  Size/request:	130673673 bytes

Response time histogram:
  2.141 [1]	|
  2.391 [5]	|■■
  2.640 [6]	|■■■
  2.889 [3]	|■
  3.139 [68]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.388 [89]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  3.637 [3]	|■
  3.887 [0]	|
  4.136 [0]	|
  4.386 [2]	|■
  4.635 [18]	|■■■■■■■■


Latency distribution:
  10% in 2.9832 secs
  25% in 3.0758 secs
  50% in 3.1579 secs
  75% in 3.2507 secs
  90% in 4.3569 secs
  95% in 4.4708 secs
  99% in 4.6349 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0023 secs, 2.1413 secs, 4.6349 secs
  DNS-lookup:	0.0001 secs, 0.0000 secs, 0.0010 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0002 secs
  resp wait:	0.0210 secs, 0.0120 secs, 0.0277 secs
  resp read:	3.2112 secs, 2.1214 secs, 4.5798 secs

Status code distribution:
  [200]	195 responses



eochieng@eochieng-thinkpadp16vgen1:~$ 
```


```Bash
eochieng@eochieng-thinkpadp16vgen1:~$ hey -c 100 -z 30s -m GET https://localhost:8083/catalogs/operatorhubio/all.json

Summary:
  Total:	34.4248 secs
  Slowest:	18.0017 secs
  Fastest:	16.4048 secs
  Average:	17.1534 secs
  Requests/sec:	5.8098
  
  Total data:	26134734600 bytes
  Size/request:	130673673 bytes

Response time histogram:
  16.405 [1]	|■
  16.564 [21]	|■■■■■■■■■■■■■■■■■■
  16.724 [47]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  16.884 [25]	|■■■■■■■■■■■■■■■■■■■■■
  17.044 [6]	|■■■■■
  17.203 [3]	|■■■
  17.363 [3]	|■■■
  17.523 [21]	|■■■■■■■■■■■■■■■■■■
  17.682 [32]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■
  17.842 [28]	|■■■■■■■■■■■■■■■■■■■■■■■■
  18.002 [13]	|■■■■■■■■■■■


Latency distribution:
  10% in 16.5490 secs
  25% in 16.6720 secs
  50% in 17.0549 secs
  75% in 17.6498 secs
  90% in 17.7859 secs
  95% in 17.8847 secs
  99% in 17.9945 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0341 secs, 16.4048 secs, 18.0017 secs
  DNS-lookup:	0.0006 secs, 0.0000 secs, 0.0025 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0006 secs
  resp wait:	0.0442 secs, 0.0093 secs, 0.1246 secs
  resp read:	17.0747 secs, 16.3778 secs, 17.8972 secs

Status code distribution:
  [200]	200 responses



eochieng@eochieng-thinkpadp16vgen1:~$
```
