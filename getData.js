function onRequest (request, response, modules) {
  // 获取用户表全部信息打印

  //获取DB对象
  var db = modules.oData;
  //查询所有数据，此处可以传where
  db.find({
    "table": "_User",
  }, function (err, data) {
    response.send(data);
  });

}