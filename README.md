# LXG - Run Desktop Apps on LXC Containers

LXG is an alternative to `distrobox` that runs directly on LXC containers, providing seamless desktop integration for GUI applications. While Distrobox focuses on integrating applications from Docker, Podman, or Lilipod containers with the host, LXG takes a different approach by using LXC system containers.

We chose LXC because by default it provides persistency and a complete Linux environments with features such as `systemd`, `init`, package manager, SSH, networking, and background services. LXG builds on this foundation by adding only the missing desktop integration layer, making applications running in LXC containers feel like native applications on the host.

In short, LXG provides desktop integration for your LXC containers.

## Building

LXG can be built with Go. Also make sure to put the binary in the correct location:

```bash
# Build binary
go build -o bin/lxg main.go

# Move to local binaries
sudo mv bin/lxg /usr/local/bin/lxg
```

## Requirements

- LXD/LXC installed and configured.
- LXG binary at `/usr/local/bin/lxg`.
- UID >= `1000` - check with `echo $UID`.
- Container with user matching UID + sudoers.

## Usage

Start by spinning up a new container or using an existing one:

```bash
CONTAINER="my-container"

# Create container
lxc launch -p default ubuntu:26.04 $CONTAINER
lxc launch -p default images:fedora/44 $CONTAINER
lxc launch -p default images:alpine/3.24 $CONTAINER
lxc launch -p default images:archlinux $CONTAINER

# List containers
lxc list
```

To add desktop support with `lxg`, just run the setup command from host:

```bash
# Add desktop support
lxg setup $CONTAINER

# Enter shell
lxg run $CONTAINER bash
lxg run $CONTAINER sh
```

The container environment now should be ready to run desktop applications.
Install and run the applications that you like.

That is it! Run `lxg help` for additional commands. 

## Desktop Entry

You can create desktop entries to directly access the program from host desktop. Here is a sample file to add Google Chrome from an LXG container:

```bash
cat << EOF > ~/.local/share/applications/lxg-google-chrome.desktop
[Desktop Entry]
Name=Google Chrome ($CONTAINER)
Comment=Launch Chrome inside LXD container seamlessly
Exec=lxg run $CONTAINER google-chrome 
Icon=google-chrome
StartupWMClass=google-chrome
StartupNotify=true
Terminal=false
Type=Application
Categories=Development;
EOF
```

## Container Requirements

The following packages are required inside the container to run desktop applications properly. When installing desktop applications from inside the container, they may be included as dependencies automatically. Otherwise, install them with the package manager:

- X11: `dbus-x11 xauth`
- Wayland: `libwayland-client0 qtwayland5`
- Mesa/GPU: `libgl1-mesa-dri libglx-mesa0 mesa-vulkan-drivers`
- Pipewire: `pipewire pipewire-pulse wireplumber`
- Secrets: `gnome-keyring libsecret`
- XDG: `xdg-utils xdg-desktop-portal`
- Gnome: `xdg-desktop-portal-gtk`
- KDE: `xdg-desktop-portal-kde kwayland`
- Cosmic: `xdg-desktop-portal-cosmic`

NOTE: Package names above are for Debian/Ubuntu based containers. Names may vary on Arch, Fedora, or Alpine.