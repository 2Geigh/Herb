<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ohana | Surf The Indie Web</title>
    <link rel="stylesheet" href="./assets/styles/index.css">
</head>

<body>
    <h1>Ohana</h1>
    <h2>Surf the <em><a href="/purpose">living</a></em> web.</h2>

    <form action="search" method="get">
        <input type="text" maxlength="255" required>
        <input id="searchButton" type="submit" value="Go">
    </form>

    <footer>
        Happy <?php echo strtolower(date('l', time())); ?>!
    </footer>
</body>


</html>