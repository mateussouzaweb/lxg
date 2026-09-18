# LXG - Run Desktop Apps on LXC Containers

LXG is an alternative to `distrobox` that runs directly on LXC containers, providing seamless desktop integration for GUI applications. While Distrobox focuses on integrating applications from Docker, Podman, or Lilipod containers with the host, LXG takes a different approach by using LXC system containers.

We chose LXC because it provides persistent, complete Linux environments with features such as `systemd`, `init`, SSH, networking, and background services. LXG builds on this foundation by adding the missing desktop integration layer, making applications running in LXC containers feel like native applications on the host.

In short, LXG provides desktop integration for your LXC containers.

## Usage

```bash
lxg run ubuntu chrome
```

## Building

```bash
go build -o bin/lxg main.go
sudo mv bin/lxg /usr/local/bin/lxg
sudo chmod +x /usr/local/bin/lxg
```