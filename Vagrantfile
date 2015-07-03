# -*- mode: ruby -*-
# vi: set ft=ruby :

Vagrant.configure("2") do |config|
  config.vm.network :private_network, ip: "192.168.33.99"
  config.vm.network :forwarded_port, guest: 22, host: 2299

  config.vbguest.no_remote = true
  config.vbguest.auto_update = false

  config.vm.define 'trusty' do |instance|
    instance.vm.box = 'ubuntu/trusty64'
  end

  config.vm.define 'precise' do |instance|
    instance.vm.box = 'ubuntu/precise64'
  end

  config.vm.synced_folder ".", "/vagrant"

end
