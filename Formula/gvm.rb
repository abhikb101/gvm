class Gvm < Formula
  desc "nvm for Git identities. Switch between multiple GitHub accounts with ease."
  homepage "https://github.com/abhikb101/gvm"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/abhikb101/gvm/releases/download/v#{version}/gvm_darwin_arm64.tar.gz"
    end
    on_intel do
      url "https://github.com/abhikb101/gvm/releases/download/v#{version}/gvm_darwin_amd64.tar.gz"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/abhikb101/gvm/releases/download/v#{version}/gvm_linux_arm64.tar.gz"
    end
    on_intel do
      url "https://github.com/abhikb101/gvm/releases/download/v#{version}/gvm_linux_amd64.tar.gz"
    end
  end

  def install
    bin.install "gvm"
  end

  test do
    system "#{bin}/gvm", "--version"
  end
end
