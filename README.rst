FIO Exporter
==============

About
-------

Run `fio <https://github.com/axboe/fio>`_  as below and make the periodical reporting results consumable for Prometheus:

::

  <path to fio binary> <path to fio job file> \
    --group_reporting --status-interval=<interval> \
    --output-format=terse --terse-version=3

Please refer to `fio latest document <https://fio.readthedocs.io/en/latest/fio_doc.html>`_ for option introduction.

Requirements
-------------

- **fio** should be installed;
- **a fio job file** should be defined;

Supported Platform
--------------------

- Linux
- Windows

Usage
-------


::

  git clone https://github.com/kckecheng/fio_exporter.git
  cd fio_exporter
  go build -v
  ./fio_exporter --help
  ./fio_exporter -p fio -j /mnt1/job1.ini -i 30 -l 8080

Notes
-------

- **filename/directory** should be defined with absolute paths, otherwise, files will be created under the same directory as this exporter;
- Option **--path/-p** is optional if the fio executable binary is searchable (under **PATH**);
- Since the reporting results are generated periodically based on option **--status-interval**, Prometheus should scrape this exporter with the same interval;
