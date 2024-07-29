@echo off

%~dp0\..\pc_monitor.exe stop
%~dp0\..\pc_monitor.exe uninstall
sc delete HydrateNowService
pause