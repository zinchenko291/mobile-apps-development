# For build

```
sudo apt update
sudo apt install linux-headers-$(uname -r) build-essential
```

# Build and Run

```
make
sudo insmod myrand.ko
head /dev/myrand
```