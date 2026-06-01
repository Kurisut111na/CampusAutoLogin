/****************************************************************************
** Meta object code from reading C++ file 'main_window.h'
**
** Created by: The Qt Meta Object Compiler version 68 (Qt 6.7.0)
**
** WARNING! All changes made in this file will be lost!
*****************************************************************************/

#include "../../../../src/ui/main_window.h"
#include <QtGui/qtextcursor.h>
#include <QtCore/qmetatype.h>

#include <QtCore/qtmochelpers.h>

#include <memory>


#include <QtCore/qxptype_traits.h>
#if !defined(Q_MOC_OUTPUT_REVISION)
#error "The header file 'main_window.h' doesn't include <QObject>."
#elif Q_MOC_OUTPUT_REVISION != 68
#error "This file was generated using the moc from 6.7.0. It"
#error "cannot be used with the include files from this version of Qt."
#error "(The moc has changed too much.)"
#endif

#ifndef Q_CONSTINIT
#define Q_CONSTINIT
#endif

QT_WARNING_PUSH
QT_WARNING_DISABLE_DEPRECATED
QT_WARNING_DISABLE_GCC("-Wuseless-cast")
namespace {

#ifdef QT_MOC_HAS_STRINGDATA
struct qt_meta_stringdata_CLASSMainWindowENDCLASS_t {};
constexpr auto qt_meta_stringdata_CLASSMainWindowENDCLASS = QtMocHelpers::stringData(
    "MainWindow",
    "onLoginClicked",
    "",
    "onReconnectClicked",
    "onRefreshIp",
    "onRefreshNetwork",
    "onAdvancedSettings",
    "onLoginSuccess",
    "onLoginFailed",
    "error",
    "onConnectionLost",
    "onConnectionRestored",
    "onConnectionAlive",
    "onTrayActivated",
    "QSystemTrayIcon::ActivationReason",
    "reason",
    "onAutoLoginTriggered",
    "onGatewayReady",
    "gateway",
    "onGatewayProbeFailed",
    "onGatewayChanged",
    "onNetworkTypeChanged",
    "NetworkType",
    "oldType",
    "newType",
    "onNewLogEntry",
    "LogEntry",
    "entry",
    "onReconnectCooldownTimeout"
);
#else  // !QT_MOC_HAS_STRINGDATA
#error "qtmochelpers.h not found or too old."
#endif // !QT_MOC_HAS_STRINGDATA
} // unnamed namespace

Q_CONSTINIT static const uint qt_meta_data_CLASSMainWindowENDCLASS[] = {

 // content:
      12,       // revision
       0,       // classname
       0,    0, // classinfo
      18,   14, // methods
       0,    0, // properties
       0,    0, // enums/sets
       0,    0, // constructors
       0,       // flags
       0,       // signalCount

 // slots: name, argc, parameters, tag, flags, initial metatype offsets
       1,    0,  122,    2, 0x08,    1 /* Private */,
       3,    0,  123,    2, 0x08,    2 /* Private */,
       4,    0,  124,    2, 0x08,    3 /* Private */,
       5,    0,  125,    2, 0x08,    4 /* Private */,
       6,    0,  126,    2, 0x08,    5 /* Private */,
       7,    0,  127,    2, 0x08,    6 /* Private */,
       8,    1,  128,    2, 0x08,    7 /* Private */,
      10,    0,  131,    2, 0x08,    9 /* Private */,
      11,    0,  132,    2, 0x08,   10 /* Private */,
      12,    0,  133,    2, 0x08,   11 /* Private */,
      13,    1,  134,    2, 0x08,   12 /* Private */,
      16,    0,  137,    2, 0x08,   14 /* Private */,
      17,    1,  138,    2, 0x08,   15 /* Private */,
      19,    0,  141,    2, 0x08,   17 /* Private */,
      20,    1,  142,    2, 0x08,   18 /* Private */,
      21,    2,  145,    2, 0x08,   20 /* Private */,
      25,    1,  150,    2, 0x08,   23 /* Private */,
      28,    0,  153,    2, 0x08,   25 /* Private */,

 // slots: parameters
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void, QMetaType::QString,    9,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void,
    QMetaType::Void, 0x80000000 | 14,   15,
    QMetaType::Void,
    QMetaType::Void, QMetaType::QString,   18,
    QMetaType::Void,
    QMetaType::Void, QMetaType::QString,   18,
    QMetaType::Void, 0x80000000 | 22, 0x80000000 | 22,   23,   24,
    QMetaType::Void, 0x80000000 | 26,   27,
    QMetaType::Void,

       0        // eod
};

Q_CONSTINIT const QMetaObject MainWindow::staticMetaObject = { {
    QMetaObject::SuperData::link<QMainWindow::staticMetaObject>(),
    qt_meta_stringdata_CLASSMainWindowENDCLASS.offsetsAndSizes,
    qt_meta_data_CLASSMainWindowENDCLASS,
    qt_static_metacall,
    nullptr,
    qt_incomplete_metaTypeArray<qt_meta_stringdata_CLASSMainWindowENDCLASS_t,
        // Q_OBJECT / Q_GADGET
        QtPrivate::TypeAndForceComplete<MainWindow, std::true_type>,
        // method 'onLoginClicked'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onReconnectClicked'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onRefreshIp'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onRefreshNetwork'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onAdvancedSettings'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onLoginSuccess'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onLoginFailed'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<const QString &, std::false_type>,
        // method 'onConnectionLost'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onConnectionRestored'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onConnectionAlive'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onTrayActivated'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<QSystemTrayIcon::ActivationReason, std::false_type>,
        // method 'onAutoLoginTriggered'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onGatewayReady'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<const QString &, std::false_type>,
        // method 'onGatewayProbeFailed'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        // method 'onGatewayChanged'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<const QString &, std::false_type>,
        // method 'onNetworkTypeChanged'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<NetworkType, std::false_type>,
        QtPrivate::TypeAndForceComplete<NetworkType, std::false_type>,
        // method 'onNewLogEntry'
        QtPrivate::TypeAndForceComplete<void, std::false_type>,
        QtPrivate::TypeAndForceComplete<const LogEntry &, std::false_type>,
        // method 'onReconnectCooldownTimeout'
        QtPrivate::TypeAndForceComplete<void, std::false_type>
    >,
    nullptr
} };

void MainWindow::qt_static_metacall(QObject *_o, QMetaObject::Call _c, int _id, void **_a)
{
    if (_c == QMetaObject::InvokeMetaMethod) {
        auto *_t = static_cast<MainWindow *>(_o);
        (void)_t;
        switch (_id) {
        case 0: _t->onLoginClicked(); break;
        case 1: _t->onReconnectClicked(); break;
        case 2: _t->onRefreshIp(); break;
        case 3: _t->onRefreshNetwork(); break;
        case 4: _t->onAdvancedSettings(); break;
        case 5: _t->onLoginSuccess(); break;
        case 6: _t->onLoginFailed((*reinterpret_cast< std::add_pointer_t<QString>>(_a[1]))); break;
        case 7: _t->onConnectionLost(); break;
        case 8: _t->onConnectionRestored(); break;
        case 9: _t->onConnectionAlive(); break;
        case 10: _t->onTrayActivated((*reinterpret_cast< std::add_pointer_t<QSystemTrayIcon::ActivationReason>>(_a[1]))); break;
        case 11: _t->onAutoLoginTriggered(); break;
        case 12: _t->onGatewayReady((*reinterpret_cast< std::add_pointer_t<QString>>(_a[1]))); break;
        case 13: _t->onGatewayProbeFailed(); break;
        case 14: _t->onGatewayChanged((*reinterpret_cast< std::add_pointer_t<QString>>(_a[1]))); break;
        case 15: _t->onNetworkTypeChanged((*reinterpret_cast< std::add_pointer_t<NetworkType>>(_a[1])),(*reinterpret_cast< std::add_pointer_t<NetworkType>>(_a[2]))); break;
        case 16: _t->onNewLogEntry((*reinterpret_cast< std::add_pointer_t<LogEntry>>(_a[1]))); break;
        case 17: _t->onReconnectCooldownTimeout(); break;
        default: ;
        }
    }
}

const QMetaObject *MainWindow::metaObject() const
{
    return QObject::d_ptr->metaObject ? QObject::d_ptr->dynamicMetaObject() : &staticMetaObject;
}

void *MainWindow::qt_metacast(const char *_clname)
{
    if (!_clname) return nullptr;
    if (!strcmp(_clname, qt_meta_stringdata_CLASSMainWindowENDCLASS.stringdata0))
        return static_cast<void*>(this);
    return QMainWindow::qt_metacast(_clname);
}

int MainWindow::qt_metacall(QMetaObject::Call _c, int _id, void **_a)
{
    _id = QMainWindow::qt_metacall(_c, _id, _a);
    if (_id < 0)
        return _id;
    if (_c == QMetaObject::InvokeMetaMethod) {
        if (_id < 18)
            qt_static_metacall(this, _c, _id, _a);
        _id -= 18;
    } else if (_c == QMetaObject::RegisterMethodArgumentMetaType) {
        if (_id < 18)
            *reinterpret_cast<QMetaType *>(_a[0]) = QMetaType();
        _id -= 18;
    }
    return _id;
}
QT_WARNING_POP
