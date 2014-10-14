go-server-commander
===================

Simple script to execute commands on multiple servers through ssh.

Usage: server-commander -i *servers_file* -c *commands_file*

 * servers_file: contains the servers you wish to run batch commands on.
 * commands_file ( optional ): contains additional commands ( provided on the spot and not be default ).

The output of the commands is written to a txt file.