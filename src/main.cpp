#include <QApplication>

#include "ui/main_window.h"
#include "logger.h"

int main(int argc, char* argv[])
{
    QApplication app(argc, argv);
    app.setApplicationName("CampusAutoLogin");
    app.setApplicationVersion("1.0.0");
    app.setOrganizationName("CampusAutoLogin");
    app.setQuitOnLastWindowClosed(false);

    Logger::instance().info("Application started");

    MainWindow mainWindow;
    if (!mainWindow.shouldStartMinimized()) {
        mainWindow.show();
    }

    int result = app.exec();

    Logger::instance().info("Application exiting");
    return result;
}