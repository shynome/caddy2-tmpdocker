## v1.2.1

### Fix

- fix: set value to new copy struct not the pointer struct
- 修复: 未使用引用赋值, 导致值没有设置成功

## v1.2.0

### Break Change

for better code

- field `wait` -> `keep_alive`

## v1.1.0

### Break Change

for better code

- field `freeze_timeout` -> `wait`(WaitingTimeBeforeStop)
- field `wake_timeout` -> `scale_timeout`

## v1.0.1

### FIX

- fix: nil pointer problem in stop docker service

## v0.0.4

### ADD

- add check timer break sign controller

### FIX

- freeze_timeout is must greater than 1m not 5m

## v0.0.3

### FIX

- fix update last active time fail
