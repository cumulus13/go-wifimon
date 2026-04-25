class Wifimon < Formula
  desc "Real-time Wi-Fi terminal monitor with Growl notifications"
  homepage "https://github.com/cumulus13/go-wifimon"
  version "@VERSION@"
  if OS.mac?
    if Hardware::CPU.arm?
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_darwin_arm64.zip"
      sha256 "@DARWIN_ARM64_SHA256@"
    else
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_darwin_amd64.zip"
      sha256 "@DARWIN_AMD64_SHA256@"
    end
  elsif OS.linux?
    if Hardware::CPU.arm?
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_linux_arm64.zip"
      sha256 "@LINUX_ARM64_SHA256@"
    else
      url "https://github.com/cumulus13/go-wifimon/releases/download/v#{version}/wifimon_#{version}_linux_amd64.zip"
      sha256 "@LINUX_AMD64_SHA256@"
    end
  end

  def install
    bin.install "wifimon"
    pkgshare.install "assets"
  end

  test do
    assert_match "Usage:", shell_output("#{bin}/wifimon --help")
  end
end
