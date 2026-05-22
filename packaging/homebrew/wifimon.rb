# This file is managed by GoReleaser.
# Manual edits will be overwritten on the next release.
# Source: https://github.com/cumulus13/go-wifimon
class Wifimon < Formula
  desc "Real-time Wi-Fi terminal monitor with Growl notifications"
  homepage "https://github.com/cumulus13/go-wifimon"
  version "0.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_darwin_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    else
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_darwin_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_linux_arm64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    else
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_linux_amd64.tar.gz"
      sha256 "0000000000000000000000000000000000000000000000000000000000000000"
    end
  end

  def install
    bin.install "wifimon"
    (pkgshare/"assets").install Dir["assets/*"]
  end

  test do
    system "#{bin}/wifimon", "--version"
  end
end
