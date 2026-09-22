# LXG - Run Desktop Apps on LXC Containers

LXG is an alternative to `distrobox` that runs directly on LXC containers, providing seamless desktop integration for GUI applications. While Distrobox focuses on integrating applications from Docker, Podman, or Lilipod containers with the host, LXG takes a different approach by using LXC system containers.

We chose LXC because by default it provides persistency and a complete Linux environments with features such as `systemd`, `init`, package manager, SSH, networking, and background services. LXG builds on this foundation by adding only the missing desktop integration layer, making applications running in LXC containers feel like native applications on the host.

In short, LXG provides desktop integration for your LXC containers.

## Project Status

This project works, but still are in the early stages of development and will present bugs or unexpected behaviors. My ultimate goal is to have the following types of container setup:

- **Isolated containers:** these are the most isolated containers, sharing only key services: GPU, audio and display. Designed for untrusted applications at the cost of no desktop integration for things like notifications, screen sharing and home folder access. In this mode, you will need to install and run a browser inside the container to open links with `xdg-open` for example.

- **Integrated containers:** containers with more deeper sharing, including GPU, audio, display and desktop integration (D-Bus) for things like notifications, screen sharing and home folder access (home folder in secondary location, since host and container users are explicitly not the same). In this mode, you can open links with `xdg-open` in the container and they will be opened in your host default browser for example.

- **Privileged containers:** provide access to the full desktop of your host, including everything that is possible such as home folder and even the root folder. This is the closest to `distrobox` experience model, but requires that both host and container have the same username.

See the table below for a more specific overview:

-- | Isolated | Integrated | Privileged
--- | --- | --- | ---
Privileged | no | no | yes
GPU | shared | shared | shared
Wayland | shared | shared | shared
X11 | shared | shared | shared
Pipewire | shared | shared | shared
Pulse | shared | shared | shared
Network | isolated | isolated | shared
IPC | isolated | isolated | shared
PID | isolated | isolated | mostly shared
D-Bus | isolated | integrated | shared
UID | must match | must match | must match
User | isolated | integrated | shared - must match
`$HOME` | isolated | integrated | shared - must match
Runtime | isolated | partially isolated | shared
Bridge | unavailable | integrated | not necessary

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
- Package `xdg-dbus-proxy` installed on host.

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

The following packages are required inside the container to run desktop applications properly. When installing desktop applications from inside the container, they may be included as dependencies automatically. Otherwise, install them with the package manager.

NOTE: Package names below are for Debian/Ubuntu based containers and names may vary on Arch, Fedora, or Alpine.

- Mesa/GPU: `mesa-vulkan-drivers vulkan-tools`
- Wayland: `wayland-utils`
- X11: `mesa-utils x11-utils x11-xserver-utils`
- XDG: `xdg-utils xdg-user-dirs dbus-bin dbus-user-session`
- PipeWire: `pipewire-bin pipewire-alsa pulseaudio-utils alsa-utils`
- Secrets: `libsecret-1-0 libsecret-tools`
- Fonts: `fontconfig fonts-liberation fonts-dejavu fonts-ubuntu fonts-noto fonts-roboto fonts-open-sans fonts-firacode`
- Others: `zenity`

When using fully isolated containers, the following packages are also required:

- X11: `dbus-x11`
- XDG / Portals:`xdg-desktop-portal xdg-desktop-portal-gtk`
- XDG / KDE: `xdg-desktop-portal-kde`
- XDG / GNOME: `xdg-desktop-portal-gnome`
- Secrets: `gnome-keyring`

When installing Pipewire, is also important to stop the service inside the container to avoid conflict with host:

```bash
sudo systemctl --global mask wireplumber.service \
    pipewire.socket pipewire.service \
    pipewire-pulse.socket pipewire-pulse.service

systemctl --user stop wireplumber.service \
    pipewire.socket pipewire.service \
    pipewire-pulse.socket pipewire-pulse.service
```