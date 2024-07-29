@echo off

%~dp0\..\hydrate_pc.exe stop
%~dp0\..\hydrate_pc.exe uninstall
sc delete HydrateNowService
pause