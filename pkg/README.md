Packaging & Deployment
----------------------

In order to build packages and upload to Package Cloud, please install the following requirements and run the make task.

[Package Cloud Command Line Client](https://packagecloud.io/docs#cli_install)

```
$ gem install package_cloud
```

[FPM](https://github.com/jordansissel/fpm)

```
$ gem install fpm
```

Building package

```
$ make package
```

*NOTE: you will be prompted for Package Cloud credentials.*

Testing
-------

```
$ git clone https://github.com/foomo/contentserver.git
$ cd contentserver
$ make test
```

Contributing
------------

In lieu of a formal styleguide, take care to maintain the existing coding style. Add unit tests and examples for any new or changed functionality.

1.	Fork it
2.	Create your feature branch (`git checkout -b my-new-feature`\)
3.	Commit your changes (`git commit -am 'Add some feature'`\)
4.	Push to the branch (`git push origin my-new-feature`\)
5.	Create new Pull Request
