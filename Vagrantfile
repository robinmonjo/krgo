# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
  config.vm.box = "trusty64"
  config.vm.box_url = "https://cloud-images.ubuntu.com/vagrant/trusty/current/trusty-server-cloudimg-amd64-vagrant-disk1.box"

  config.vm.synced_folder ".", "/vagrant", disabled: true
  config.vm.synced_folder ".", "/vagrant/krgo"

  config.vm.provision "shell", path: "setup.sh"

  config.vm.network :public_network

  config.vm.provider "virtualbox" do |v|
    v.memory = 2048
  end

end
