# VPS deployment

Host: `ssh cctv-vps`.

- systemd unit: `video-service.service` (template in this directory).
- Releases: `/opt/video-service/releases/`; active release: `/opt/video-service/current`.
- Private config: `/etc/video-service/cameras.json`, root:video-service, mode 640.
- HTTP: port 80. WebRTC uses UDP ports allocated by Pion.
- Startup follows the existing `video-network.service` for the camera network route.

```sh
ssh cctv-vps 'systemctl status video-service --no-pager'
ssh cctv-vps 'journalctl -u video-service -n 50 --no-pager'
ssh cctv-vps 'systemctl restart video-service'
```

Build the frontend with `npm run build --prefix front`. Build the backend for
Linux amd64 with `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o video-server ./cmd/video-server`.
A release contains the executable and `front/dist/`. Keep private configs outside
release directories. Point `current` at the new release and restart the systemd unit.

The current application has no authentication; the HTTP interface and camera
signaling endpoint are reachable by anyone with network access to port 80.

Deployed release: `20260909-01` at `http://46.173.214.126/`.
Verified: service enabled/running, both startup probes succeed, public catalog
responds, browser receives the low stream at 1280×720.

Release `20260909-02`: frontend updated to a per-camera player grid with explicit
click-to-play. Existing backend binary and `/etc/video-service/cameras.json` preserved.
Previous release remains in `/opt/video-service/releases/20260909-01`.

Release `20260909-03`: fullscreen switches the selected player to high quality;
leaving fullscreen restores low quality. Manual quality selectors removed.
Backend and production camera configuration preserved.
